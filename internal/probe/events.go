package probe

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Типы событий events.jsonl. Каждая строка — {"seq":N,"ts":"...","kind":"...", ...поля события}.

type evSession struct {
	Profile string `json:"profile"`
	Version string `json:"probe_version,omitempty"`
	Browser string `json:"browser,omitempty"`
	Note    string `json:"note,omitempty"`
}

type evTarget struct {
	Event    string `json:"event"` // attached | detached | created | destroyed
	TargetID string `json:"target_id"`
	Type     string `json:"type,omitempty"`
	URL      string `json:"url,omitempty"`
	OpenerID string `json:"opener_id,omitempty"`
	ParentID string `json:"parent_target_id,omitempty"`
}

type evNavigation struct {
	TargetID     string `json:"target_id"`
	FrameID      string `json:"frame_id"`
	MainFrame    bool   `json:"main_frame"`
	SameDocument bool   `json:"same_document"`
	URL          string `json:"url"`
}

type evRequest struct {
	ID            string            `json:"id"`
	TargetID      string            `json:"target_id"`
	FrameID       string            `json:"frame_id,omitempty"`
	ResourceType  string            `json:"resource_type"`
	Method        string            `json:"method"`
	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          string            `json:"body,omitempty"`
	BodySize      int               `json:"body_size,omitempty"`
	BodyTruncated bool              `json:"body_truncated,omitempty"`
	BodyError     string            `json:"body_error,omitempty"`
	Initiator     string            `json:"initiator,omitempty"`
	InitiatorURL  string            `json:"initiator_url,omitempty"`
	RedirectOf    string            `json:"redirect_of,omitempty"`
}

type evResponse struct {
	ID                string            `json:"id"`
	URL               string            `json:"url"`
	Status            int               `json:"status,omitempty"`
	StatusText        string            `json:"status_text,omitempty"`
	MimeType          string            `json:"mime_type,omitempty"`
	Headers           map[string]string `json:"headers,omitempty"`
	HeadersRaw        map[string]string `json:"headers_raw,omitempty"`
	RequestHeadersRaw map[string]string `json:"request_headers_raw,omitempty"`
	Body              string            `json:"body,omitempty"`
	BodySize          int               `json:"body_size,omitempty"`
	BodyTruncated     bool              `json:"body_truncated,omitempty"`
	BodyBinary        bool              `json:"body_binary_omitted,omitempty"`
	BodyError         string            `json:"body_error,omitempty"`
	FromCache         bool              `json:"from_cache,omitempty"`
	Redirect          bool              `json:"redirect,omitempty"`
	Error             string            `json:"error,omitempty"`
	DurationMS        int64             `json:"duration_ms"`
}

type evExtraHeaders struct {
	ID                string            `json:"id"`
	HeadersRaw        map[string]string `json:"headers_raw,omitempty"`
	RequestHeadersRaw map[string]string `json:"request_headers_raw,omitempty"`
}

type evAction struct {
	N        int             `json:"n"`
	Action   string          `json:"action"`
	TargetID string          `json:"target_id"`
	URL      string          `json:"url"`
	Element  *actionElement  `json:"element,omitempty"`
	ValueLen *int            `json:"value_len,omitempty"`
	Checked  *bool           `json:"checked,omitempty"`
	Selected []string        `json:"selected,omitempty"`
	Files    *int            `json:"files,omitempty"`
	Form     *actionFormInfo `json:"form,omitempty"`
}

type actionFormInfo struct {
	Action string `json:"action,omitempty"`
	Method string `json:"method,omitempty"`
}

type actionElement struct {
	Tag    string `json:"tag"`
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Type   string `json:"type,omitempty"`
	Role   string `json:"role,omitempty"`
	Label  string `json:"label,omitempty"`
	Text   string `json:"text,omitempty"`
	Class  string `json:"class,omitempty"`
	Href   string `json:"href,omitempty"`
	TestID string `json:"testid,omitempty"`
}

type evMark struct {
	N        int      `json:"n"`
	Name     string   `json:"name"`
	TargetID string   `json:"target_id,omitempty"`
	URL      string   `json:"url,omitempty"`
	HTML     string   `json:"html,omitempty"`
	PNG      string   `json:"png,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

// eventWriter пишет events.jsonl построчно и сразу сбрасывает на диск: при аварийном завершении
// записанное остаётся в рабочем каталоге.
type eventWriter struct {
	mu     sync.Mutex
	f      *os.File
	bw     *bufio.Writer
	seq    int
	closed bool
	err    error
}

func newEventWriter(path string) (*eventWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	return &eventWriter{f: f, bw: bufio.NewWriterSize(f, 256<<10)}, nil
}

const tsLayout = "2006-01-02T15:04:05.000Z07:00"

func (w *eventWriter) write(ts time.Time, kind string, payload any) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || w.err != nil {
		return
	}
	body, err := json.Marshal(payload)
	if err != nil {
		w.err = err
		return
	}
	w.seq++
	head, _ := json.Marshal(struct {
		Seq  int    `json:"seq"`
		TS   string `json:"ts"`
		Kind string `json:"kind"`
	}{w.seq, ts.Format(tsLayout), kind})
	line := head[:len(head)-1]
	if len(body) > 2 {
		line = append(line, ',')
		line = append(line, body[1:]...)
	} else {
		line = append(line, '}')
	}
	line = append(line, '\n')
	if _, err := w.bw.Write(line); err != nil {
		w.err = err
		return
	}
	if err := w.bw.Flush(); err != nil {
		w.err = err
	}
}

func (w *eventWriter) close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return w.err
	}
	w.closed = true
	if err := w.bw.Flush(); err != nil && w.err == nil {
		w.err = err
	}
	if err := w.f.Close(); err != nil && w.err == nil {
		w.err = err
	}
	return w.err
}
