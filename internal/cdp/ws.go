package cdp

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Минимальный клиент WebSocket (RFC 6455) — ровно то, что нужно для DevTools-протокола Chrome:
// текстовые сообщения, фрагментация, ping/pong, close. Своя реализация вместо сторонней библиотеки,
// чтобы у бинаря не было зависимостей (CLAUDE.md: стандартная библиотека в приоритете).

const (
	opContinuation = 0x0
	opText         = 0x1
	opBinary       = 0x2
	opClose        = 0x8
	opPing         = 0x9
	opPong         = 0xA

	// maxMessageSize ограничивает одно сообщение: скриншоты и тела ответов в base64 бывают
	// по несколько мегабайт, но сотни мегабайт — уже признак ошибки.
	maxMessageSize = 256 << 20

	wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
)

// ErrClosed — соединение закрыто (штатно или со стороны браузера).
var ErrClosed = errors.New("cdp: соединение закрыто")

type wsConn struct {
	conn net.Conn
	br   *bufio.Reader

	writeMu sync.Mutex
}

// dialWS открывает WebSocket по адресу вида ws://127.0.0.1:port/devtools/browser/<id>.
func dialWS(ctx context.Context, rawURL string) (*wsConn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("cdp: адрес websocket: %w", err)
	}
	if u.Scheme != "ws" {
		return nil, fmt.Errorf("cdp: поддерживается только ws://, получено %q", u.Scheme)
	}
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", u.Host)
	if err != nil {
		return nil, fmt.Errorf("cdp: подключение к %s: %w", u.Host, err)
	}
	ok := false
	defer func() {
		if !ok {
			_ = conn.Close()
		}
	}()
	if dl, has := ctx.Deadline(); has {
		_ = conn.SetDeadline(dl)
	}

	keyRaw := make([]byte, 16)
	if _, err := rand.Read(keyRaw); err != nil {
		return nil, err
	}
	key := base64.StdEncoding.EncodeToString(keyRaw)

	path := u.RequestURI()
	req := "GET " + path + " HTTP/1.1\r\n" +
		"Host: " + u.Host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"
	if _, err := io.WriteString(conn, req); err != nil {
		return nil, fmt.Errorf("cdp: рукопожатие: %w", err)
	}
	br := bufio.NewReaderSize(conn, 64<<10)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodGet})
	if err != nil {
		return nil, fmt.Errorf("cdp: ответ на рукопожатие: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return nil, fmt.Errorf("cdp: рукопожатие отклонено: %s", resp.Status)
	}
	if !strings.EqualFold(resp.Header.Get("Upgrade"), "websocket") {
		return nil, errors.New("cdp: сервер не перешёл на websocket")
	}
	if resp.Header.Get("Sec-WebSocket-Accept") != acceptKey(key) {
		return nil, errors.New("cdp: неверный Sec-WebSocket-Accept")
	}
	_ = conn.SetDeadline(time.Time{})
	ok = true
	return &wsConn{conn: conn, br: br}, nil
}

func acceptKey(key string) string {
	h := sha1.Sum([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(h[:])
}

// writeFrame пишет один кадр; клиент обязан маскировать данные.
func (c *wsConn) writeFrame(op byte, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return writeFrame(c.conn, op, payload, true)
}

func writeFrame(w io.Writer, op byte, payload []byte, masked bool) error {
	hdr := make([]byte, 0, 14)
	hdr = append(hdr, 0x80|op)
	maskBit := byte(0)
	if masked {
		maskBit = 0x80
	}
	n := len(payload)
	switch {
	case n < 126:
		hdr = append(hdr, maskBit|byte(n))
	case n <= 0xFFFF:
		hdr = append(hdr, maskBit|126)
		hdr = binary.BigEndian.AppendUint16(hdr, uint16(n))
	default:
		hdr = append(hdr, maskBit|127)
		hdr = binary.BigEndian.AppendUint64(hdr, uint64(n))
	}
	data := payload
	if masked {
		var mask [4]byte
		if _, err := rand.Read(mask[:]); err != nil {
			return err
		}
		hdr = append(hdr, mask[:]...)
		data = make([]byte, n)
		for i := range payload {
			data[i] = payload[i] ^ mask[i%4]
		}
	}
	buf := append(hdr, data...)
	_, err := w.Write(buf)
	return err
}

// readMessage возвращает очередное текстовое или бинарное сообщение, собирая фрагменты
// и отвечая на ping. На close-кадр отвечает close и возвращает ErrClosed.
func (c *wsConn) readMessage() ([]byte, error) {
	var msg []byte
	started := false
	for {
		fin, op, payload, err := readFrame(c.br)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return nil, ErrClosed
			}
			return nil, err
		}
		switch op {
		case opPing:
			if err := c.writeFrame(opPong, payload); err != nil {
				return nil, err
			}
			continue
		case opPong:
			continue
		case opClose:
			_ = c.writeFrame(opClose, nil)
			return nil, ErrClosed
		case opText, opBinary:
			if started {
				return nil, errors.New("cdp: новое сообщение внутри фрагментированного")
			}
			started = true
			msg = payload
		case opContinuation:
			if !started {
				return nil, errors.New("cdp: продолжение без начала сообщения")
			}
			msg = append(msg, payload...)
		default:
			return nil, fmt.Errorf("cdp: неизвестный opcode %d", op)
		}
		if len(msg) > maxMessageSize {
			return nil, errors.New("cdp: сообщение больше допустимого размера")
		}
		if fin {
			return msg, nil
		}
	}
}

func readFrame(r io.Reader) (fin bool, op byte, payload []byte, err error) {
	var h [2]byte
	if _, err = io.ReadFull(r, h[:]); err != nil {
		return
	}
	fin = h[0]&0x80 != 0
	op = h[0] & 0x0F
	masked := h[1]&0x80 != 0
	n := uint64(h[1] & 0x7F)
	switch n {
	case 126:
		var b [2]byte
		if _, err = io.ReadFull(r, b[:]); err != nil {
			return
		}
		n = uint64(binary.BigEndian.Uint16(b[:]))
	case 127:
		var b [8]byte
		if _, err = io.ReadFull(r, b[:]); err != nil {
			return
		}
		n = binary.BigEndian.Uint64(b[:])
	}
	if n > maxMessageSize {
		err = errors.New("cdp: кадр больше допустимого размера")
		return
	}
	var mask [4]byte
	if masked {
		if _, err = io.ReadFull(r, mask[:]); err != nil {
			return
		}
	}
	payload = make([]byte, n)
	if _, err = io.ReadFull(r, payload); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			err = io.EOF
		}
		return
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return
}

func (c *wsConn) close() error {
	_ = c.writeFrame(opClose, nil)
	return c.conn.Close()
}
