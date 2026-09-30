// Package cdp — минимальный клиент Chrome DevTools Protocol поверх websocket
// в режиме плоских сессий (Target.attachToTarget с flatten: true).
package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// Event — событие протокола. SessionID пуст для событий уровня браузера.
type Event struct {
	SessionID string
	Method    string
	Params    json.RawMessage
}

// Error — ошибка, которую вернул сам браузер в ответ на команду.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return fmt.Sprintf("cdp: %s (код %d)", e.Message, e.Code) }

type message struct {
	ID        int64           `json:"id,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *Error          `json:"error,omitempty"`
}

type reply struct {
	result json.RawMessage
	err    error
}

// Conn — соединение с браузером. Call безопасен для одновременного вызова из нескольких горутин.
// События складываются в неограниченную очередь: читатель соединения никогда не ждёт
// обработчика, поэтому обработчик событий может сам вызывать Call без взаимной блокировки.
type Conn struct {
	ws     *wsConn
	nextID atomic.Int64

	mu      sync.Mutex
	pending map[int64]chan reply
	err     error

	queue  []Event
	qCond  *sync.Cond
	events chan Event

	done chan struct{}
}

// Dial подключается к websocket-адресу браузера.
func Dial(ctx context.Context, wsURL string) (*Conn, error) {
	ws, err := dialWS(ctx, wsURL)
	if err != nil {
		return nil, err
	}
	c := &Conn{
		ws:      ws,
		pending: make(map[int64]chan reply),
		events:  make(chan Event),
		done:    make(chan struct{}),
	}
	c.qCond = sync.NewCond(&c.mu)
	go c.readLoop()
	go c.pump()
	return c, nil
}

// Events — поток событий. Закрывается, когда соединение закрыто и очередь выбрана.
func (c *Conn) Events() <-chan Event { return c.events }

// Done закрывается, когда соединение закрыто.
func (c *Conn) Done() <-chan struct{} { return c.done }

// Err — причина закрытия соединения (после Done).
func (c *Conn) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Close закрывает соединение.
func (c *Conn) Close() error {
	err := c.ws.close()
	<-c.done
	return err
}

// Call отправляет команду и ждёт ответа. sessionID пуст для команд уровня браузера.
func (c *Conn) Call(ctx context.Context, sessionID, method string, params any) (json.RawMessage, error) {
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("cdp: %s: параметры: %w", method, err)
		}
		raw = b
	}
	id := c.nextID.Add(1)
	ch := make(chan reply, 1)

	c.mu.Lock()
	if c.err != nil {
		err := c.err
		c.mu.Unlock()
		return nil, err
	}
	c.pending[id] = ch
	c.mu.Unlock()

	b, err := json.Marshal(message{ID: id, SessionID: sessionID, Method: method, Params: raw})
	if err != nil {
		c.forget(id)
		return nil, err
	}
	if err := c.ws.writeFrame(opText, b); err != nil {
		c.forget(id)
		return nil, fmt.Errorf("cdp: %s: отправка: %w", method, err)
	}
	select {
	case r := <-ch:
		if r.err != nil {
			return nil, fmt.Errorf("%s: %w", method, r.err)
		}
		return r.result, nil
	case <-ctx.Done():
		c.forget(id)
		return nil, fmt.Errorf("cdp: %s: %w", method, ctx.Err())
	}
}

// CallInto — Call с разбором результата в out.
func (c *Conn) CallInto(ctx context.Context, sessionID, method string, params, out any) error {
	res, err := c.Call(ctx, sessionID, method, params)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(res, out); err != nil {
		return fmt.Errorf("cdp: %s: разбор ответа: %w", method, err)
	}
	return nil
}

func (c *Conn) forget(id int64) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

func (c *Conn) readLoop() {
	var err error
	for {
		var data []byte
		data, err = c.ws.readMessage()
		if err != nil {
			break
		}
		var m message
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		if m.ID != 0 {
			c.mu.Lock()
			ch := c.pending[m.ID]
			delete(c.pending, m.ID)
			c.mu.Unlock()
			if ch != nil {
				if m.Error != nil {
					ch <- reply{err: m.Error}
				} else {
					ch <- reply{result: m.Result}
				}
			}
			continue
		}
		if m.Method != "" {
			c.mu.Lock()
			c.queue = append(c.queue, Event{SessionID: m.SessionID, Method: m.Method, Params: m.Params})
			c.qCond.Signal()
			c.mu.Unlock()
		}
	}
	_ = c.ws.conn.Close()
	if !errors.Is(err, ErrClosed) {
		err = fmt.Errorf("cdp: чтение: %w", err)
	}
	c.mu.Lock()
	c.err = ErrClosed
	if !errors.Is(err, ErrClosed) {
		c.err = errors.Join(ErrClosed, err)
	}
	for id, ch := range c.pending {
		ch <- reply{err: c.err}
		delete(c.pending, id)
	}
	c.qCond.Broadcast()
	c.mu.Unlock()
	close(c.done)
}

// pump перекладывает события из неограниченной очереди в канал Events.
func (c *Conn) pump() {
	defer close(c.events)
	for {
		c.mu.Lock()
		for len(c.queue) == 0 && c.err == nil {
			c.qCond.Wait()
		}
		if len(c.queue) == 0 {
			c.mu.Unlock()
			return
		}
		ev := c.queue[0]
		c.queue[0] = Event{}
		c.queue = c.queue[1:]
		c.mu.Unlock()
		c.events <- ev
	}
}
