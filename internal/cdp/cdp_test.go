package cdp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestFrameRoundTrip(t *testing.T) {
	for _, n := range []int{0, 5, 125, 126, 65535, 65536, 200000} {
		payload := bytes.Repeat([]byte("я"), n/2+1)[:n]
		for _, masked := range []bool{true, false} {
			var buf bytes.Buffer
			if err := writeFrame(&buf, opText, payload, masked); err != nil {
				t.Fatal(err)
			}
			fin, op, got, err := readFrame(&buf)
			if err != nil || !fin || op != opText || !bytes.Equal(got, payload) {
				t.Fatalf("n=%d masked=%v: fin=%v op=%d err=%v равны=%v", n, masked, fin, op, err, bytes.Equal(got, payload))
			}
		}
	}
}

// fakeBrowser — минимальный websocket-сервер DevTools для проверки клиента без Chrome.
type fakeBrowser struct {
	ln net.Listener
}

func newFakeBrowser(t *testing.T, handle func(br *bufio.Reader, conn net.Conn)) *fakeBrowser {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		br := bufio.NewReader(conn)
		req, err := http.ReadRequest(br)
		if err != nil {
			return
		}
		key := req.Header.Get("Sec-WebSocket-Key")
		_, _ = conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + acceptKey(key) + "\r\n\r\n"))
		handle(br, conn)
	}()
	return &fakeBrowser{ln: ln}
}

func (f *fakeBrowser) url() string { return "ws://" + f.ln.Addr().String() + "/devtools/browser/x" }

func TestConnCallEventsPingFragments(t *testing.T) {
	pong := make(chan string, 1)
	fb := newFakeBrowser(t, func(br *bufio.Reader, conn net.Conn) {
		// Сначала ping — клиент обязан ответить pong (он может прийти и после первой команды).
		_ = writeFrame(conn, opPing, []byte("p"), false)
		for {
			_, op, payload, err := readFrame(br)
			if err != nil || op == opClose {
				return
			}
			if op == opPong {
				pong <- string(payload)
				continue
			}
			var m message
			_ = json.Unmarshal(payload, &m)
			// Событие фрагментами: текст + продолжение.
			ev := []byte(`{"method":"Test.event","sessionId":"S1","params":{"n":1}}`)
			_ = writeRawFrame(conn, false, opText, ev[:10])
			_ = writeRawFrame(conn, true, opContinuation, ev[10:])
			switch m.Method {
			case "Test.ok":
				b, _ := json.Marshal(message{ID: m.ID, Result: json.RawMessage(`{"v":42}`)})
				_ = writeFrame(conn, opText, b, false)
			case "Test.fail":
				b, _ := json.Marshal(message{ID: m.ID, Error: &Error{Code: -32000, Message: "нет такого"}})
				_ = writeFrame(conn, opText, b, false)
			}
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := Dial(ctx, fb.url())
	if err != nil {
		t.Fatal(err)
	}
	var out struct{ V int }
	if err := c.CallInto(ctx, "S1", "Test.ok", map[string]any{"a": 1}, &out); err != nil || out.V != 42 {
		t.Fatalf("Test.ok: %v %v", out, err)
	}
	var cerr *Error
	if _, err := c.Call(ctx, "", "Test.fail", nil); !errors.As(err, &cerr) || cerr.Code != -32000 {
		t.Fatalf("Test.fail: %v", err)
	}
	for i := 0; i < 2; i++ {
		select {
		case ev := <-c.Events():
			if ev.Method != "Test.event" || ev.SessionID != "S1" || !strings.Contains(string(ev.Params), `"n":1`) {
				t.Fatalf("событие: %+v", ev)
			}
		case <-ctx.Done():
			t.Fatal("нет события")
		}
	}
	select {
	case p := <-pong:
		if p != "p" {
			t.Fatalf("pong: %q", p)
		}
	case <-ctx.Done():
		t.Fatal("нет pong")
	}
	_ = c.Close()
	if _, err := c.Call(ctx, "", "Test.ok", nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("после закрытия: %v", err)
	}
	// Канал событий закрывается после закрытия соединения.
	for range c.Events() {
	}
}

func TestPendingCallFailsWhenBrowserCloses(t *testing.T) {
	fb := newFakeBrowser(t, func(br *bufio.Reader, conn net.Conn) {
		_, _, _, _ = readFrame(br) // команда пришла, браузер «упал»
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := Dial(ctx, fb.url())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Call(ctx, "", "Test.hang", nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("ждали ErrClosed, получили %v", err)
	}
	<-c.Done()
}

func TestDialRejectsBadHandshake(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		_, _ = http.ReadRequest(bufio.NewReader(conn))
		_, _ = conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nSec-WebSocket-Accept: wrong\r\n\r\n"))
		_ = conn.Close()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := Dial(ctx, "ws://"+ln.Addr().String()+"/"); err == nil {
		t.Fatal("рукопожатие с неверным Accept принято")
	}
}

func writeRawFrame(conn net.Conn, fin bool, op byte, payload []byte) error {
	var buf bytes.Buffer
	if err := writeFrame(&buf, op, payload, false); err != nil {
		return err
	}
	b := buf.Bytes()
	if !fin {
		b[0] &^= 0x80
	}
	_, err := conn.Write(b)
	return err
}
