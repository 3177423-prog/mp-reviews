// Package probe — запись того, что делает кабинет маркетплейса, пока пользователь работает в нём
// руками (SPEC §11, этап 0). Пробник только наблюдает: не кликает, не заполняет и не отправляет
// ничего в кабинете.
package probe

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/3177423-prog/mp-reviews/internal/cdp"
)

// Version — версия формата записи пробника.
const Version = "0.1"

// recordedTypes — типы ресурсов, которые записываются: запросы XHR/fetch и навигации.
var recordedTypes = map[string]bool{"XHR": true, "Fetch": true, "Document": true, "EventSource": true}

// Options — настройки записи.
type Options struct {
	// Profile — имя профиля кабинета (wb-ms, oz-cr, …), попадает в имя файла и в summary.md.
	Profile string
	// OutDir — каталог для zip и временного рабочего каталога.
	OutDir string
	// BodyLimit — сколько байт тела запроса или ответа сохранять (по умолчанию 1 МиБ).
	BodyLimit int
	// LinkWindow — запрос в пределах этого времени после действия считается вызванным им (по умолчанию 10 с).
	LinkWindow time.Duration
	Logger     *slog.Logger
	// Now — часы (для тестов).
	Now func() time.Time
}

// Recorder подписывается на события браузера и пишет их в рабочий каталог.
type Recorder struct {
	conn  *cdp.Conn
	opts  Options
	log   *slog.Logger
	start time.Time
	stamp string

	workDir string
	w       *eventWriter

	loopDone chan struct{}

	mu        sync.Mutex
	finished  bool
	sessions  map[string]*session
	pending   map[string]*pendingReq
	orphans   map[string]*extraInfo
	completed map[string]string // ключ запроса → id, для поздних ExtraInfo
	doneOrder []string
	reqSeq    int
	actions   int
	marks     int
	active    string // сессия страницы, где было последнее действие или навигация
	idx       index
}

type session struct {
	id       string
	targetID string
	typ      string
	url      string
	page     string // сессия вкладки, к которой относится цель (для iframe — родительская)
}

type pendingReq struct {
	id        string
	sessionID string
	targetID  string
	started   time.Time
	url       string
	status    int
	statusTxt string
	mime      string
	headers   map[string]string
	fromCache bool
	extra     extraInfo
	item      *reqItem
}

type extraInfo struct {
	reqHeaders  map[string]string
	respHeaders map[string]string
	at          time.Time
}

// New создаёт рабочий каталог и файл событий. Подписка на браузер — в Start.
func New(conn *cdp.Conn, opts Options) (*Recorder, error) {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.BodyLimit <= 0 {
		opts.BodyLimit = 1 << 20
	}
	if opts.LinkWindow <= 0 {
		opts.LinkWindow = 10 * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.DiscardHandler)
	}
	if opts.OutDir == "" {
		opts.OutDir = "."
	}
	if opts.Profile == "" {
		return nil, errors.New("probe: не задан профиль")
	}
	start := opts.Now()
	stamp := start.Format("20060102-150405")
	if err := os.MkdirAll(opts.OutDir, 0o700); err != nil {
		return nil, fmt.Errorf("probe: каталог результата: %w", err)
	}
	// Прежние записи не перезаписываются: при совпадении имени добавляется номер.
	var workDir string
	for i := 1; ; i++ {
		if i > 1 {
			stamp = start.Format("20060102-150405") + "-" + strconv.Itoa(i)
		}
		workDir = filepath.Join(opts.OutDir, "probe-"+opts.Profile+"-"+stamp)
		_, zipErr := os.Stat(workDir + ".zip")
		err := os.Mkdir(workDir, 0o700)
		if err == nil && errors.Is(zipErr, os.ErrNotExist) {
			break
		}
		if err == nil {
			_ = os.Remove(workDir)
		} else if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("probe: рабочий каталог: %w", err)
		}
		if i >= 100 {
			return nil, errors.New("probe: не удалось выбрать имя рабочего каталога")
		}
	}
	if err := os.Mkdir(filepath.Join(workDir, "marks"), 0o700); err != nil {
		return nil, fmt.Errorf("probe: рабочий каталог: %w", err)
	}
	w, err := newEventWriter(filepath.Join(workDir, "events.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("probe: events.jsonl: %w", err)
	}
	r := &Recorder{
		conn:      conn,
		opts:      opts,
		log:       opts.Logger,
		start:     start,
		stamp:     stamp,
		workDir:   workDir,
		w:         w,
		loopDone:  make(chan struct{}),
		sessions:  make(map[string]*session),
		pending:   make(map[string]*pendingReq),
		orphans:   make(map[string]*extraInfo),
		completed: make(map[string]string),
	}
	r.idx.profile = opts.Profile
	r.idx.start = start
	return r, nil
}

// WorkDir — рабочий каталог записи (удаляется после сборки zip).
func (r *Recorder) WorkDir() string { return r.workDir }

// Start включает автоподключение ко всем вкладкам, iframe и воркерам и запускает обработку событий.
// Новые цели ставятся на паузу до подписки на сеть, поэтому первые запросы новой вкладки не теряются.
func (r *Recorder) Start(ctx context.Context) error {
	var ver struct {
		Product string `json:"product"`
	}
	_ = r.conn.CallInto(ctx, "", "Browser.getVersion", nil, &ver)
	r.w.write(r.opts.Now(), "session_start", evSession{Profile: r.opts.Profile, Version: Version, Browser: ver.Product})

	go r.loop()

	if _, err := r.conn.Call(ctx, "", "Target.setDiscoverTargets", map[string]any{"discover": true}); err != nil {
		return err
	}
	_, err := r.conn.Call(ctx, "", "Target.setAutoAttach", map[string]any{
		"autoAttach": true, "waitForDebuggerOnStart": true, "flatten": true,
	})
	return err
}

// Done закрывается, когда соединение с браузером закрыто и все события обработаны.
func (r *Recorder) Done() <-chan struct{} { return r.loopDone }

// Navigate открывает адрес в текущей вкладке — только стартовая страница, по флагу -url.
func (r *Recorder) Navigate(ctx context.Context, url string) error {
	sid, err := r.waitPage(ctx)
	if err != nil {
		return err
	}
	_, err = r.conn.Call(ctx, sid, "Page.navigate", map[string]any{"url": url})
	return err
}

func (r *Recorder) waitPage(ctx context.Context) (string, error) {
	for {
		r.mu.Lock()
		sid := r.active
		r.mu.Unlock()
		if sid != "" {
			return sid, nil
		}
		select {
		case <-ctx.Done():
			return "", errors.New("probe: нет открытой вкладки")
		case <-r.loopDone:
			return "", errors.New("probe: браузер закрыт")
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (r *Recorder) loop() {
	defer close(r.loopDone)
	for ev := range r.conn.Events() {
		r.handle(ev)
	}
}

func (r *Recorder) handle(ev cdp.Event) {
	r.mu.Lock()
	finished := r.finished
	r.mu.Unlock()
	if finished {
		return
	}
	switch ev.Method {
	case "Target.attachedToTarget":
		r.onAttached(ev)
	case "Target.detachedFromTarget":
		r.onDetached(ev)
	case "Target.targetCreated", "Target.targetDestroyed":
		r.onTargetLifecycle(ev)
	case "Network.requestWillBeSent":
		r.onRequest(ev)
	case "Network.requestWillBeSentExtraInfo", "Network.responseReceivedExtraInfo":
		r.onExtraInfo(ev)
	case "Network.responseReceived":
		r.onResponse(ev)
	case "Network.loadingFinished":
		r.onFinished(ev, "")
	case "Network.loadingFailed":
		var p struct {
			ErrorText string `json:"errorText"`
			Canceled  bool   `json:"canceled"`
		}
		_ = json.Unmarshal(ev.Params, &p)
		msg := p.ErrorText
		if p.Canceled {
			msg = "отменён: " + msg
		}
		if msg == "" {
			msg = "ошибка загрузки"
		}
		r.onFinished(ev, msg)
	case "Page.frameNavigated":
		var p struct {
			Frame struct {
				ID       string `json:"id"`
				ParentID string `json:"parentId"`
				URL      string `json:"url"`
				Fragment string `json:"urlFragment"`
			} `json:"frame"`
		}
		if json.Unmarshal(ev.Params, &p) == nil {
			r.onNavigated(ev.SessionID, p.Frame.ID, p.Frame.URL+p.Frame.Fragment, p.Frame.ParentID == "", false)
		}
	case "Page.navigatedWithinDocument":
		var p struct {
			FrameID string `json:"frameId"`
			URL     string `json:"url"`
		}
		if json.Unmarshal(ev.Params, &p) == nil {
			r.onNavigated(ev.SessionID, p.FrameID, p.URL, r.isMainFrame(ev.SessionID, p.FrameID), true)
		}
	case "Runtime.bindingCalled":
		r.onBinding(ev)
	}
}

func (r *Recorder) callCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}

// --- цели ---

func (r *Recorder) onAttached(ev cdp.Event) {
	var p struct {
		SessionID  string `json:"sessionId"`
		TargetInfo struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
			URL      string `json:"url"`
			OpenerID string `json:"openerId"`
		} `json:"targetInfo"`
		Waiting bool `json:"waitingForDebugger"`
	}
	if json.Unmarshal(ev.Params, &p) != nil || p.SessionID == "" {
		return
	}
	ti := p.TargetInfo
	s := &session{id: p.SessionID, targetID: ti.TargetID, typ: ti.Type, url: ti.URL}
	r.mu.Lock()
	parent := r.sessions[ev.SessionID]
	parentTarget := ""
	switch {
	case ti.Type == "page":
		s.page = s.id
	case parent != nil:
		s.page = parent.page
		parentTarget = parent.targetID
	}
	r.sessions[s.id] = s
	r.mu.Unlock()
	r.w.write(r.opts.Now(), "target", evTarget{Event: "attached", TargetID: ti.TargetID, Type: ti.Type,
		URL: RedactURL(ti.URL), OpenerID: ti.OpenerID, ParentID: parentTarget})
	if ti.Type == "page" && ti.OpenerID != "" {
		r.mu.Lock()
		r.idx.add(item{ts: r.opts.Now(), kind: itemTab, url: RedactURL(ti.URL)})
		r.mu.Unlock()
	}

	ctx, cancel := r.callCtx()
	defer cancel()
	sid := p.SessionID
	call := func(method string, params any) {
		if _, err := r.conn.Call(ctx, sid, method, params); err != nil {
			r.log.Debug("настройка цели", "type", ti.Type, "method", method, "err", err)
		}
	}
	switch ti.Type {
	case "page", "iframe":
		call("Network.enable", map[string]any{
			"maxTotalBufferSize": 200 << 20, "maxResourceBufferSize": 20 << 20, "maxPostDataSize": r.opts.BodyLimit,
		})
		call("Page.enable", nil)
		// Без Runtime.enable браузер не присылает Runtime.bindingCalled (проверено на Chromium 141).
		call("Runtime.enable", nil)
		call("Runtime.addBinding", map[string]any{"name": bindingName, "executionContextName": actionWorld})
		call("Page.addScriptToEvaluateOnNewDocument", map[string]any{
			"source": actionScript, "worldName": actionWorld, "runImmediately": true,
		})
		if !p.Waiting {
			r.installInExistingFrames(ctx, sid)
		}
	case "worker", "shared_worker", "service_worker":
		// Воркеры расширений Chrome (chrome-extension://) к кабинету не относятся.
		if strings.HasPrefix(ti.URL, "http://") || strings.HasPrefix(ti.URL, "https://") {
			call("Network.enable", map[string]any{"maxPostDataSize": r.opts.BodyLimit})
		}
	}
	if ti.Type != "browser" {
		call("Target.setAutoAttach", map[string]any{"autoAttach": true, "waitForDebuggerOnStart": true, "flatten": true})
	}
	if p.Waiting {
		call("Runtime.runIfWaitingForDebugger", nil)
	}
	// Текущей вкладка становится только после подписки на события: иначе стартовая навигация
	// (флаг -url) могла бы начаться раньше, чем включён Page.
	if ti.Type == "page" {
		r.mu.Lock()
		if _, alive := r.sessions[sid]; alive && (r.active == "" || ti.OpenerID != "") {
			r.active = sid
		}
		r.mu.Unlock()
	}
}

// installInExistingFrames ставит журнал действий в уже открытый документ (вкладка была открыта
// до запуска записи): runImmediately поддерживается не всеми версиями Chrome.
func (r *Recorder) installInExistingFrames(ctx context.Context, sid string) {
	var tree struct {
		FrameTree frameTree `json:"frameTree"`
	}
	if err := r.conn.CallInto(ctx, sid, "Page.getFrameTree", nil, &tree); err != nil {
		return
	}
	var walk func(ft frameTree)
	walk = func(ft frameTree) {
		var w struct {
			ContextID int `json:"executionContextId"`
		}
		if r.conn.CallInto(ctx, sid, "Page.createIsolatedWorld", map[string]any{
			"frameId": ft.Frame.ID, "worldName": actionWorld,
		}, &w) == nil && w.ContextID != 0 {
			_, _ = r.conn.Call(ctx, sid, "Runtime.evaluate", map[string]any{
				"expression": actionScript, "contextId": w.ContextID,
			})
		}
		for _, c := range ft.Children {
			walk(c)
		}
	}
	walk(tree.FrameTree)
}

type frameTree struct {
	Frame struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"frame"`
	Children []frameTree `json:"childFrames"`
}

func (r *Recorder) onDetached(ev cdp.Event) {
	var p struct {
		SessionID string `json:"sessionId"`
	}
	if json.Unmarshal(ev.Params, &p) != nil {
		return
	}
	r.mu.Lock()
	s := r.sessions[p.SessionID]
	delete(r.sessions, p.SessionID)
	var orphaned []*pendingReq
	for k, pr := range r.pending {
		if pr.sessionID == p.SessionID {
			orphaned = append(orphaned, pr)
			delete(r.pending, k)
		}
	}
	if r.active == p.SessionID {
		r.active = ""
		for _, o := range r.sessions {
			if o.typ == "page" {
				r.active = o.id
				break
			}
		}
	}
	r.mu.Unlock()
	for _, pr := range orphaned {
		r.emitResponse(pr, "", false, "цель закрыта до завершения запроса", r.opts.Now())
	}
	if s != nil {
		r.w.write(r.opts.Now(), "target", evTarget{Event: "detached", TargetID: s.targetID, Type: s.typ})
	}
}

func (r *Recorder) onTargetLifecycle(ev cdp.Event) {
	var p struct {
		TargetInfo struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
			URL      string `json:"url"`
			OpenerID string `json:"openerId"`
		} `json:"targetInfo"`
		TargetID string `json:"targetId"`
	}
	if json.Unmarshal(ev.Params, &p) != nil {
		return
	}
	if ev.Method == "Target.targetCreated" {
		if p.TargetInfo.Type != "page" {
			return
		}
		r.w.write(r.opts.Now(), "target", evTarget{Event: "created", TargetID: p.TargetInfo.TargetID,
			Type: p.TargetInfo.Type, URL: RedactURL(p.TargetInfo.URL), OpenerID: p.TargetInfo.OpenerID})
		return
	}
	r.w.write(r.opts.Now(), "target", evTarget{Event: "destroyed", TargetID: p.TargetID})
}

func (r *Recorder) isMainFrame(sid, frameID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.sessions[sid]
	return s != nil && s.typ == "page" && s.targetID == frameID
}

func (r *Recorder) onNavigated(sid, frameID, url string, main, sameDoc bool) {
	now := r.opts.Now()
	r.mu.Lock()
	s := r.sessions[sid]
	if s == nil {
		r.mu.Unlock()
		return
	}
	main = main && s.typ == "page"
	if main {
		s.url = url
		r.active = s.page
		r.idx.add(item{ts: now, kind: itemNav, url: RedactURL(url), sameDoc: sameDoc})
	}
	target := s.targetID
	r.mu.Unlock()
	r.w.write(now, "navigation", evNavigation{TargetID: target, FrameID: frameID, MainFrame: main,
		SameDocument: sameDoc, URL: RedactURL(url)})
}

// --- действия пользователя ---

func (r *Recorder) onBinding(ev cdp.Event) {
	var p struct {
		Name    string `json:"name"`
		Payload string `json:"payload"`
	}
	if json.Unmarshal(ev.Params, &p) != nil || p.Name != bindingName {
		return
	}
	var a struct {
		Action     string         `json:"action"`
		TS         float64        `json:"ts"`
		URL        string         `json:"url"`
		El         *actionElement `json:"el"`
		ValueLen   *int           `json:"value_len"`
		Checked    *bool          `json:"checked"`
		Selected   []string       `json:"selected"`
		Files      *int           `json:"files"`
		FormAction string         `json:"form_action"`
		FormMethod string         `json:"form_method"`
	}
	if json.Unmarshal([]byte(p.Payload), &a) != nil || a.Action == "" {
		return
	}
	ts := r.opts.Now()
	if a.TS > 0 {
		ts = time.UnixMilli(int64(a.TS))
	}
	if a.El != nil {
		a.El.Href = RedactURL(a.El.Href)
		a.El.Text = redactText(a.El.Text)
		a.El.Label = redactText(a.El.Label)
	}
	out := evAction{Action: a.Action, URL: RedactURL(a.URL), Element: a.El, ValueLen: a.ValueLen,
		Checked: a.Checked, Selected: a.Selected, Files: a.Files}
	if a.FormAction != "" || a.FormMethod != "" {
		out.Form = &actionFormInfo{Action: RedactURL(a.FormAction), Method: a.FormMethod}
	}
	r.mu.Lock()
	r.actions++
	out.N = r.actions
	if s := r.sessions[ev.SessionID]; s != nil {
		out.TargetID = s.targetID
		if s.page != "" {
			r.active = s.page
		}
	}
	r.idx.add(item{ts: ts, kind: itemAction, n: out.N, desc: describeAction(out)})
	r.mu.Unlock()
	r.w.write(ts, "action", out)
}

// --- сеть ---

func reqKey(sid, requestID string) string { return sid + "|" + requestID }

type cdpRequest struct {
	URL             string            `json:"url"`
	URLFragment     string            `json:"urlFragment"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	PostData        string            `json:"postData"`
	HasPostData     bool              `json:"hasPostData"`
	PostDataEntries []struct {
		Bytes string `json:"bytes"`
	} `json:"postDataEntries"`
}

type cdpResponse struct {
	URL               string            `json:"url"`
	Status            int               `json:"status"`
	StatusText        string            `json:"statusText"`
	Headers           map[string]string `json:"headers"`
	MimeType          string            `json:"mimeType"`
	FromDiskCache     bool              `json:"fromDiskCache"`
	FromServiceWorker bool              `json:"fromServiceWorker"`
	FromPrefetchCache bool              `json:"fromPrefetchCache"`
}

func (r *Recorder) onRequest(ev cdp.Event) {
	var p struct {
		RequestID        string       `json:"requestId"`
		Request          cdpRequest   `json:"request"`
		WallTime         float64      `json:"wallTime"`
		Type             string       `json:"type"`
		FrameID          string       `json:"frameId"`
		RedirectResponse *cdpResponse `json:"redirectResponse"`
		Initiator        struct {
			Type string `json:"type"`
			URL  string `json:"url"`
		} `json:"initiator"`
	}
	if json.Unmarshal(ev.Params, &p) != nil {
		return
	}
	key := reqKey(ev.SessionID, p.RequestID)
	now := r.opts.Now()
	started := now
	if p.WallTime > 0 {
		sec := int64(p.WallTime)
		started = time.Unix(sec, int64((p.WallTime-float64(sec))*1e9))
	}

	// Перенаправление приходит тем же requestId: закрываем предыдущий шаг цепочки.
	redirectOf := ""
	r.mu.Lock()
	prev := r.pending[key]
	if prev != nil && p.RedirectResponse != nil {
		delete(r.pending, key)
		prev.applyResponse(*p.RedirectResponse)
		redirectOf = prev.id
	}
	r.mu.Unlock()
	if prev != nil && p.RedirectResponse != nil {
		r.emitResponse(prev, "", true, "", now)
	}

	if !recordedTypes[p.Type] {
		return
	}

	req := p.Request
	fullURL := req.URL + req.URLFragment
	body, bodyErr := r.requestBody(ev.SessionID, p.RequestID, req)
	ctype := headerValue(req.Headers, "Content-Type")
	redBody := RedactBody(body, ctype)
	reqKeys := topLevelKeys(redBody, ctype)
	stored, truncated := truncateUTF8(redBody, r.opts.BodyLimit)

	r.mu.Lock()
	r.reqSeq++
	id := "r" + strconv.Itoa(r.reqSeq)
	s := r.sessions[ev.SessionID]
	target := ""
	if s != nil {
		target = s.targetID
	}
	pr := &pendingReq{id: id, sessionID: ev.SessionID, targetID: target, started: started, url: fullURL}
	if o := r.orphans[key]; o != nil {
		pr.extra = *o
		delete(r.orphans, key)
	}
	it := &reqItem{id: id, method: req.Method, url: RedactURL(fullURL), resType: p.Type, reqKeys: append(queryKeys(fullURL), reqKeys...)}
	pr.item = it
	r.pending[key] = pr
	r.idx.add(item{ts: started, kind: itemRequest, req: it})
	r.mu.Unlock()

	r.w.write(started, "request", evRequest{
		ID: id, TargetID: target, FrameID: p.FrameID, ResourceType: p.Type, Method: req.Method,
		URL: RedactURL(fullURL), Headers: RedactHeaders(req.Headers),
		Body: stored, BodySize: len(body), BodyTruncated: truncated, BodyError: bodyErr,
		Initiator: p.Initiator.Type, InitiatorURL: RedactURL(p.Initiator.URL), RedirectOf: redirectOf,
	})
}

// requestBody достаёт тело запроса: из события, из postDataEntries или отдельным вызовом.
func (r *Recorder) requestBody(sid, requestID string, req cdpRequest) (string, string) {
	if req.PostData != "" {
		return req.PostData, ""
	}
	if len(req.PostDataEntries) > 0 {
		var b strings.Builder
		ok := true
		for _, e := range req.PostDataEntries {
			d, err := base64.StdEncoding.DecodeString(e.Bytes)
			if err != nil {
				ok = false
				break
			}
			b.Write(d)
		}
		if ok && b.Len() > 0 {
			return b.String(), ""
		}
	}
	if !req.HasPostData {
		return "", ""
	}
	ctx, cancel := r.callCtx()
	defer cancel()
	var res struct {
		PostData string `json:"postData"`
	}
	if err := r.conn.CallInto(ctx, sid, "Network.getRequestPostData", map[string]any{"requestId": requestID}, &res); err != nil {
		return "", "тело недоступно: " + err.Error()
	}
	return res.PostData, ""
}

func (r *Recorder) onExtraInfo(ev cdp.Event) {
	var p struct {
		RequestID string            `json:"requestId"`
		Headers   map[string]string `json:"headers"`
	}
	if json.Unmarshal(ev.Params, &p) != nil {
		return
	}
	isReq := ev.Method == "Network.requestWillBeSentExtraInfo"
	key := reqKey(ev.SessionID, p.RequestID)
	now := r.opts.Now()
	r.mu.Lock()
	if pr := r.pending[key]; pr != nil {
		if isReq {
			pr.extra.reqHeaders = p.Headers
		} else {
			pr.extra.respHeaders = p.Headers
		}
		r.mu.Unlock()
		return
	}
	if id, ok := r.completed[key]; ok {
		r.mu.Unlock()
		out := evExtraHeaders{ID: id}
		if isReq {
			out.RequestHeadersRaw = RedactHeaders(p.Headers)
		} else {
			out.HeadersRaw = RedactHeaders(p.Headers)
		}
		r.w.write(now, "extra_headers", out)
		return
	}
	// ExtraInfo может прийти раньше requestWillBeSent — придержим. Для незаписываемых типов
	// ресурсов (картинки, скрипты) такие записи устаревают и выбрасываются.
	o := r.orphans[key]
	if o == nil {
		if len(r.orphans) > 2000 {
			for k, v := range r.orphans {
				if now.Sub(v.at) > 30*time.Second {
					delete(r.orphans, k)
				}
			}
		}
		o = &extraInfo{at: now}
		r.orphans[key] = o
	}
	if isReq {
		o.reqHeaders = p.Headers
	} else {
		o.respHeaders = p.Headers
	}
	r.mu.Unlock()
}

func (pr *pendingReq) applyResponse(resp cdpResponse) {
	pr.status = resp.Status
	pr.statusTxt = resp.StatusText
	pr.mime = resp.MimeType
	pr.headers = resp.Headers
	pr.fromCache = resp.FromDiskCache || resp.FromServiceWorker || resp.FromPrefetchCache
	if resp.URL != "" {
		pr.url = resp.URL
	}
}

func (r *Recorder) onResponse(ev cdp.Event) {
	var p struct {
		RequestID string      `json:"requestId"`
		Response  cdpResponse `json:"response"`
	}
	if json.Unmarshal(ev.Params, &p) != nil {
		return
	}
	r.mu.Lock()
	if pr := r.pending[reqKey(ev.SessionID, p.RequestID)]; pr != nil {
		pr.applyResponse(p.Response)
	}
	r.mu.Unlock()
}

func (r *Recorder) onFinished(ev cdp.Event, failure string) {
	var p struct {
		RequestID string `json:"requestId"`
	}
	if json.Unmarshal(ev.Params, &p) != nil {
		return
	}
	key := reqKey(ev.SessionID, p.RequestID)
	r.mu.Lock()
	pr := r.pending[key]
	delete(r.pending, key)
	if pr != nil {
		r.completed[key] = pr.id
		r.doneOrder = append(r.doneOrder, key)
		if len(r.doneOrder) > 5000 {
			delete(r.completed, r.doneOrder[0])
			r.doneOrder = r.doneOrder[1:]
		}
	}
	r.mu.Unlock()
	if pr == nil {
		return
	}
	bodyOf := "" // тело забираем только у успешно загруженных
	if failure == "" {
		bodyOf = p.RequestID
	}
	r.emitResponse(pr, bodyOf, false, failure, r.opts.Now())
}

// emitResponse пишет событие response. requestID — непустой, если нужно забрать тело ответа.
func (r *Recorder) emitResponse(pr *pendingReq, requestID string, redirect bool, failure string, now time.Time) {
	out := evResponse{
		ID: pr.id, URL: RedactURL(pr.url), Status: pr.status, StatusText: pr.statusTxt, MimeType: pr.mime,
		Headers: RedactHeaders(pr.headers), HeadersRaw: RedactHeaders(pr.extra.respHeaders),
		RequestHeadersRaw: RedactHeaders(pr.extra.reqHeaders), FromCache: pr.fromCache,
		Redirect: redirect, Error: failure, DurationMS: now.Sub(pr.started).Milliseconds(),
	}
	var respKeys []string
	if requestID != "" {
		ctx, cancel := r.callCtx()
		var res struct {
			Body          string `json:"body"`
			Base64Encoded bool   `json:"base64Encoded"`
		}
		err := r.conn.CallInto(ctx, pr.sessionID, "Network.getResponseBody", map[string]any{"requestId": requestID}, &res)
		cancel()
		switch {
		case err != nil:
			out.BodyError = "тело недоступно: " + err.Error()
		case res.Base64Encoded:
			out.BodyBinary = true
			out.BodySize = base64.StdEncoding.DecodedLen(len(res.Body))
		default:
			red := RedactBody(res.Body, pr.mime)
			respKeys = topLevelKeys(red, pr.mime)
			out.BodySize = len(res.Body)
			out.Body, out.BodyTruncated = truncateUTF8(red, r.opts.BodyLimit)
		}
	}
	r.mu.Lock()
	if pr.item != nil {
		pr.item.status = pr.status
		pr.item.err = failure
		pr.item.respKeys = respKeys
		pr.item.done = true
		pr.item.redirect = redirect
	}
	r.mu.Unlock()
	r.w.write(now, "response", out)
}

// --- отметки шагов ---

// MarkResult — что сохранено по отметке.
type MarkResult struct {
	N      int
	HTML   string
	PNG    string
	Errors []string
}

// Mark сохраняет снимок DOM и скриншот текущей вкладки с меткой шага.
func (r *Recorder) Mark(ctx context.Context, name string) MarkResult {
	now := r.opts.Now()
	name = strings.TrimSpace(name)
	r.mu.Lock()
	r.marks++
	n := r.marks
	sid := r.active
	var s session
	if ss := r.sessions[sid]; ss != nil {
		s = *ss
	}
	r.mu.Unlock()

	base := fmt.Sprintf("%02d-%s", n, slug(name))
	res := MarkResult{N: n}
	ev := evMark{N: n, Name: name, TargetID: s.targetID, URL: RedactURL(s.url)}
	if sid == "" {
		res.Errors = append(res.Errors, "нет открытой вкладки")
	} else {
		if html, err := r.snapshot(ctx, sid); err != nil {
			res.Errors = append(res.Errors, "снимок DOM: "+err.Error())
		} else if err := os.WriteFile(filepath.Join(r.workDir, "marks", base+".html"), []byte(html), 0o600); err != nil {
			res.Errors = append(res.Errors, "снимок DOM: "+err.Error())
		} else {
			res.HTML = "marks/" + base + ".html"
		}
		if png, err := r.screenshot(ctx, sid); err != nil {
			res.Errors = append(res.Errors, "скриншот: "+err.Error())
		} else if err := os.WriteFile(filepath.Join(r.workDir, "marks", base+".png"), png, 0o600); err != nil {
			res.Errors = append(res.Errors, "скриншот: "+err.Error())
		} else {
			res.PNG = "marks/" + base + ".png"
		}
	}
	ev.HTML, ev.PNG, ev.Errors = res.HTML, res.PNG, res.Errors
	r.mu.Lock()
	r.idx.add(item{ts: now, kind: itemMark, n: n, desc: name, files: []string{res.HTML, res.PNG}})
	r.mu.Unlock()
	r.w.write(now, "mark", ev)
	return res
}

func (r *Recorder) snapshot(ctx context.Context, sid string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var tree struct {
		FrameTree frameTree `json:"frameTree"`
	}
	if err := r.conn.CallInto(ctx, sid, "Page.getFrameTree", nil, &tree); err != nil {
		return "", err
	}
	var w struct {
		ContextID int `json:"executionContextId"`
	}
	if err := r.conn.CallInto(ctx, sid, "Page.createIsolatedWorld", map[string]any{
		"frameId": tree.FrameTree.Frame.ID, "worldName": snapshotWorld,
	}, &w); err != nil {
		return "", err
	}
	var ev struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
		Exception *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := r.conn.CallInto(ctx, sid, "Runtime.evaluate", map[string]any{
		"expression": snapshotScript, "contextId": w.ContextID, "returnByValue": true,
	}, &ev); err != nil {
		return "", err
	}
	if ev.Exception != nil {
		return "", errors.New(ev.Exception.Text)
	}
	return redactText(ev.Result.Value), nil
}

func (r *Recorder) screenshot(ctx context.Context, sid string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var res struct {
		Data string `json:"data"`
	}
	if err := r.conn.CallInto(ctx, sid, "Page.captureScreenshot", map[string]any{"format": "png"}, &res); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(res.Data)
}

// --- завершение ---

// Finish дописывает незавершённые запросы, собирает summary.md и zip и удаляет рабочий каталог.
// Возвращает путь к zip.
func (r *Recorder) Finish() (string, error) {
	now := r.opts.Now()
	r.mu.Lock()
	if r.finished {
		r.mu.Unlock()
		return "", errors.New("probe: запись уже завершена")
	}
	r.finished = true
	left := make([]*pendingReq, 0, len(r.pending))
	for _, pr := range r.pending {
		left = append(left, pr)
	}
	r.pending = map[string]*pendingReq{}
	r.mu.Unlock()
	for _, pr := range left {
		r.emitResponseNoBody(pr, now)
	}
	r.w.write(now, "session_end", evSession{Profile: r.opts.Profile})
	if err := r.w.close(); err != nil {
		return "", fmt.Errorf("probe: events.jsonl: %w", err)
	}

	r.mu.Lock()
	r.idx.end = now
	summary := r.idx.render(r.opts.LinkWindow)
	r.mu.Unlock()
	if err := os.WriteFile(filepath.Join(r.workDir, "summary.md"), []byte(summary), 0o600); err != nil {
		return "", fmt.Errorf("probe: summary.md: %w", err)
	}
	zipPath := filepath.Join(r.opts.OutDir, "probe-"+r.opts.Profile+"-"+r.stamp+".zip")
	if err := zipDir(r.workDir, zipPath); err != nil {
		return "", err
	}
	if err := os.RemoveAll(r.workDir); err != nil {
		r.log.Warn("не удалось удалить рабочий каталог", "dir", r.workDir, "err", err)
	}
	return zipPath, nil
}

// emitResponseNoBody — запрос не завершился до конца записи (долгий опрос, EventSource).
func (r *Recorder) emitResponseNoBody(pr *pendingReq, now time.Time) {
	out := evResponse{ID: pr.id, URL: RedactURL(pr.url), Status: pr.status, StatusText: pr.statusTxt,
		MimeType: pr.mime, Headers: RedactHeaders(pr.headers), HeadersRaw: RedactHeaders(pr.extra.respHeaders),
		RequestHeadersRaw: RedactHeaders(pr.extra.reqHeaders), Error: "не завершён к концу записи",
		DurationMS: now.Sub(pr.started).Milliseconds()}
	r.mu.Lock()
	if pr.item != nil {
		pr.item.status = pr.status
		pr.item.err = out.Error
		pr.item.done = true
	}
	r.mu.Unlock()
	// Файл событий ещё открыт: Finish закрывает его после этого вызова.
	r.w.write(now, "response", out)
}

// --- вспомогательное ---

func headerValue(h map[string]string, name string) string {
	for k, v := range h {
		if strings.EqualFold(k, name) {
			return v
		}
	}
	return ""
}

func truncateUTF8(s string, limit int) (string, bool) {
	if len(s) <= limit {
		return s, false
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut], true
}

// slug — имя шага для имени файла: латиница (кириллица транслитерируется), цифры, '-', '_';
// до 60 символов. Имена файлов в zip — ASCII, чтобы архив одинаково открывался любым распаковщиком
// Windows; само название шага сохраняется в events.jsonl и summary.md как есть.
func slug(name string) string {
	var b strings.Builder
	dash := false
	for _, c := range strings.ToLower(name) {
		if b.Len() >= 60 {
			break
		}
		part, known := translit[c]
		if c == '_' || c == '-' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') {
			part, known = string(c), true
		}
		if known {
			b.WriteString(part)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 60 {
		out = strings.Trim(out[:60], "-")
	}
	if out == "" {
		out = "step"
	}
	return out
}

var translit = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e", 'ж': "zh", 'з': "z", 'и': "i",
	'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t",
	'у': "u", 'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch", 'ъ': "", 'ы': "y", 'ь': "",
	'э': "e", 'ю': "yu", 'я': "ya",
}
