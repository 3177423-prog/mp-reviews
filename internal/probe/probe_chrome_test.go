package probe

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/3177423-prog/mp-reviews/internal/cdp"
	"github.com/3177423-prog/mp-reviews/internal/chrome"
)

// Приёмка этапа 0 (SPEC §11): локальная страница с кнопкой и отправкой формы через fetch,
// открытая в настоящем headless Chromium. Пробник должен записать запрос с ответом, связать его
// с действием, вычистить Cookie / Authorization / ?token= и собрать zip с summary.md.

// Секреты тестовой страницы. Ни одна из этих строк не должна попасть в zip.
var testSecrets = []string{
	"SECRET-COOKIE-1", "SECRET-SETCOOKIE-2", "SECRET-QUERY-3", "SECRET-AUTH-4",
	"SECRET-HDR-5", "SECRET-RESP-6", "SECRET-BODY-7",
}

const testPage = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Тестовый кабинет</title></head>
<body>
<h1>Отзывы</h1>
<form id="complaint" action="/never">
  <select id="reason" name="reason"><option value="1">Другое</option><option value="11">Отзыв не относится к товару</option></select>
  <textarea id="text" name="text"></textarea>
  <button id="send" type="submit">Пожаловаться</button>
</form>
<div id="result"></div>
<button id="open" type="button" onclick="window.open('/popup', '_blank')">Статусы жалоб</button>
<script src="/app.js"></script>
</body></html>`

// Секреты лежат в отдельном скрипте: сам документ секретов не содержит, а скрипты
// пробником не записываются.
const testApp = `document.getElementById('complaint').addEventListener('submit', (e) => {
  e.preventDefault();
  fetch('/api/complaint?token=SECRET-QUERY-3&page=1', {
    method: 'POST',
    headers: {'Authorization': 'Bearer SECRET-AUTH-4', 'X-Session-Id': 'SECRET-HDR-5', 'Content-Type': 'application/json'},
    body: JSON.stringify({reviewId: 'abc', reason: Number(document.getElementById('reason').value),
      text: document.getElementById('text').value, csrfToken: 'SECRET-BODY-7'}),
  }).then((r) => r.json()).then((j) => { document.getElementById('result').textContent = 'OK ' + j.id; });
});`

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "sid", Value: "SECRET-COOKIE-1", Path: "/"})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, testPage)
	})
	mux.HandleFunc("GET /app.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		_, _ = io.WriteString(w, testApp)
	})
	mux.HandleFunc("POST /api/complaint", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("sid"); err != nil || c.Value != "SECRET-COOKIE-1" {
			http.Error(w, "no cookie", http.StatusForbidden)
			return
		}
		if r.Header.Get("Authorization") != "Bearer SECRET-AUTH-4" {
			http.Error(w, "no auth", http.StatusUnauthorized)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var in map[string]any
		if json.Unmarshal(body, &in) != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "SECRET-SETCOOKIE-2", Path: "/"})
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"id":42,"status":"queued","reason":%v,"access_token":"SECRET-RESP-6"}`, in["reason"])
	})
	mux.HandleFunc("GET /popup", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, `<!DOCTYPE html><html><body><div id="who"></div><script>
fetch('/api/whoami').then((r) => r.json()).then((j) => { document.getElementById('who').textContent = j.supplier; });
</script></body></html>`)
	})
	mux.HandleFunc("GET /api/whoami", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"supplier":"ИП Тест","supplierId":"b6d1"}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// findTestChrome ищет Chromium для теста: MP_REVIEWS_CHROME_PATH, PATH, затем браузеры Playwright.
func findTestChrome(t *testing.T) string {
	t.Helper()
	if p, err := chrome.FindExecutable(); err == nil {
		return p
	}
	for _, pattern := range []string{
		filepath.Join(os.Getenv("PLAYWRIGHT_BROWSERS_PATH"), "chromium-*", "chrome-linux", "chrome"),
		"/opt/pw-browsers/chromium-*/chrome-linux/chrome",
	} {
		if m, _ := filepath.Glob(pattern); len(m) > 0 {
			return m[len(m)-1]
		}
	}
	t.Skip("Chromium не найден: задайте " + chrome.EnvChromePath)
	return ""
}

// user — «пользователь»: отдельное CDP-подключение, которое кликает и печатает в странице,
// как это делал бы человек. Пробник о нём ничего не знает.
type user struct {
	t    *testing.T
	conn *cdp.Conn
	sid  string
}

func newUser(ctx context.Context, t *testing.T, wsURL string) *user {
	t.Helper()
	conn, err := cdp.Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("подключение пользователя: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	var targets struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
		} `json:"targetInfos"`
	}
	if err := conn.CallInto(ctx, "", "Target.getTargets", nil, &targets); err != nil {
		t.Fatal(err)
	}
	for _, ti := range targets.TargetInfos {
		if ti.Type != "page" {
			continue
		}
		var att struct {
			SessionID string `json:"sessionId"`
		}
		if err := conn.CallInto(ctx, "", "Target.attachToTarget", map[string]any{"targetId": ti.TargetID, "flatten": true}, &att); err != nil {
			t.Fatal(err)
		}
		return &user{t: t, conn: conn, sid: att.SessionID}
	}
	t.Fatal("нет вкладки")
	return nil
}

func (u *user) eval(ctx context.Context, expr string, out any) {
	u.t.Helper()
	var res struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
	}
	if err := u.conn.CallInto(ctx, u.sid, "Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true}, &res); err != nil {
		u.t.Fatalf("evaluate %s: %v", expr, err)
	}
	if out != nil {
		_ = json.Unmarshal(res.Result.Value, out)
	}
}

func (u *user) waitFor(ctx context.Context, expr string) {
	u.t.Helper()
	for {
		var ok bool
		u.eval(ctx, expr, &ok)
		if ok {
			return
		}
		select {
		case <-ctx.Done():
			u.t.Fatalf("не дождались: %s", expr)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// click — настоящий клик мышью по центру элемента.
func (u *user) click(ctx context.Context, selector string) {
	u.t.Helper()
	var pt struct{ X, Y float64 }
	u.eval(ctx, `(() => { const r = document.querySelector('`+selector+`').getBoundingClientRect();
		return {X: r.x + r.width / 2, Y: r.y + r.height / 2}; })()`, &pt)
	for _, typ := range []string{"mousePressed", "mouseReleased"} {
		if _, err := u.conn.Call(ctx, u.sid, "Input.dispatchMouseEvent", map[string]any{
			"type": typ, "x": pt.X, "y": pt.Y, "button": "left", "clickCount": 1,
		}); err != nil {
			u.t.Fatal(err)
		}
	}
}

func (u *user) typeText(ctx context.Context, text string) {
	u.t.Helper()
	if _, err := u.conn.Call(ctx, u.sid, "Input.insertText", map[string]any{"text": text}); err != nil {
		u.t.Fatal(err)
	}
}

func (u *user) selectOption(ctx context.Context, selector, value string) {
	u.t.Helper()
	// Выбор в выпадающем списке headless-режим мышью не открывает; значение ставим и шлём change,
	// как это делает браузер после выбора пункта.
	u.eval(ctx, `(() => { const s = document.querySelector('`+selector+`'); s.value = '`+value+`';
		s.dispatchEvent(new Event('change', {bubbles: true})); return true; })()`, nil)
}

type probeEnv struct {
	rec   *Recorder
	user  *user
	srv   *httptest.Server
	wsURL string
}

// startProbe запускает headless Chromium, пробник и «пользователя» на тестовой странице.
func startProbe(ctx context.Context, t *testing.T) probeEnv {
	t.Helper()
	exe := findTestChrome(t)
	srv := testServer(t)
	args := []string{"--disable-gpu"}
	if runtime.GOOS == "linux" {
		args = append(args, "--no-sandbox") // в контейнере тесты идут от root
	}
	browser, err := chrome.Launch(ctx, chrome.Options{
		ExecPath: exe, UserDataDir: t.TempDir(), Headless: true, ExtraArgs: args,
	})
	if err != nil {
		t.Fatalf("запуск Chromium: %v", err)
	}
	t.Cleanup(browser.Kill)

	conn, err := cdp.Dial(ctx, browser.WebSocketURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	// Штатное закрытие: после Kill дочерние процессы ещё пишут в профиль и мешают удалить TempDir.
	t.Cleanup(func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = conn.Call(closeCtx, "", "Browser.close", nil)
		browser.Wait(10 * time.Second)
	})

	rec, err := New(conn, Options{Profile: "wb-test", OutDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if err := rec.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := rec.Navigate(ctx, srv.URL+"/"); err != nil {
		t.Fatal(err)
	}
	u := newUser(ctx, t, browser.WebSocketURL)
	u.waitFor(ctx, `document.readyState === 'complete' && !!document.getElementById('send')`)
	return probeEnv{rec: rec, user: u, srv: srv, wsURL: browser.WebSocketURL}
}

func (r *Recorder) waitCompleted(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for !r.hasCompleted(path) {
		if time.Now().After(deadline) {
			t.Fatalf("пробник не записал ответ на %s", path)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestProbeRecordsLocalPageInHeadlessChromium(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	env := startProbe(ctx, t)
	rec, u := env.rec, env.user

	u.click(ctx, "#text")
	u.typeText(ctx, "Покупатель пишет не о нашем товаре")
	u.selectOption(ctx, "#reason", "11")
	u.click(ctx, "#send")
	u.waitFor(ctx, `document.getElementById('result').textContent === 'OK 42'`)

	// Ждём, пока пробник получит тело ответа.
	rec.waitCompleted(t, "/api/complaint")
	mark := rec.Mark(ctx, "после жалобы")
	if len(mark.Errors) > 0 {
		t.Fatalf("отметка: %v", mark.Errors)
	}
	zipPath, err := rec.Finish()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(rec.WorkDir()); !os.IsNotExist(err) {
		t.Errorf("рабочий каталог не удалён: %v", err)
	}

	files := readZip(t, zipPath)
	for _, name := range []string{"events.jsonl", "summary.md", "marks/01-posle-zhaloby.html", "marks/01-posle-zhaloby.png"} {
		if _, ok := files[name]; !ok {
			t.Fatalf("в zip нет %s; есть: %v", name, keys(files))
		}
	}
	if !bytes.HasPrefix(files["marks/01-posle-zhaloby.png"], []byte("\x89PNG")) {
		t.Error("скриншот — не PNG")
	}
	if !bytes.Contains(files["marks/01-posle-zhaloby.html"], []byte("Пожаловаться")) {
		t.Error("снимок DOM не содержит кнопку")
	}

	// Вычистка: ни один секрет не попал ни в один файл архива.
	for name, data := range files {
		for _, s := range testSecrets {
			if bytes.Contains(data, []byte(s)) {
				t.Errorf("секрет %s попал в %s", s, name)
			}
		}
	}

	events := parseEvents(t, files["events.jsonl"])
	req := findEvent(events, func(e map[string]any) bool {
		return e["kind"] == "request" && strings.Contains(str(e["url"]), "/api/complaint")
	})
	if req == nil {
		t.Fatal("нет события request для /api/complaint")
	}
	if got := str(req["url"]); !strings.Contains(got, "token="+Hidden+"&page=1") {
		t.Errorf("url: %s", got)
	}
	if req["method"] != "POST" || req["resource_type"] != "Fetch" {
		t.Errorf("method/type: %v %v", req["method"], req["resource_type"])
	}
	hdr := req["headers"].(map[string]any)
	if hdr["Authorization"] != Hidden || hdr["X-Session-Id"] != Hidden {
		t.Errorf("заголовки запроса: %v", hdr)
	}
	body := str(req["body"])
	for _, want := range []string{`"reason":11`, `"reviewId":"abc"`, `"text":"Покупатель пишет не о нашем товаре"`, `"csrfToken":"` + Hidden + `"`} {
		if !strings.Contains(body, want) {
			t.Errorf("тело запроса без %s: %s", want, body)
		}
	}
	id := str(req["id"])
	resp := findEvent(events, func(e map[string]any) bool { return e["kind"] == "response" && e["id"] == id })
	if resp == nil {
		t.Fatal("нет события response")
	}
	if resp["status"] != float64(200) {
		t.Errorf("status: %v", resp["status"])
	}
	rb := str(resp["body"])
	if !strings.Contains(rb, `"id":42`) || !strings.Contains(rb, `"access_token":"`+Hidden+`"`) {
		t.Errorf("тело ответа: %s", rb)
	}
	// Фактически отправленные заголовки (с Cookie) и Set-Cookie приходят в ExtraInfo —
	// в событии response или отдельным extra_headers.
	cookie, setCookie := "", ""
	for _, e := range events {
		if e["id"] != id {
			continue
		}
		if h, ok := e["request_headers_raw"].(map[string]any); ok {
			cookie = str(headerAny(h, "cookie"))
		}
		if h, ok := e["headers_raw"].(map[string]any); ok {
			setCookie = str(headerAny(h, "set-cookie"))
		}
	}
	if cookie != "sid="+Hidden {
		t.Errorf("Cookie: %q", cookie)
	}
	if !strings.HasPrefix(setCookie, "session="+Hidden) {
		t.Errorf("Set-Cookie: %q", setCookie)
	}

	// Действия пользователя записаны, запрос связан с кликом по кнопке.
	click := findEvent(events, func(e map[string]any) bool {
		el, _ := e["element"].(map[string]any)
		return e["kind"] == "action" && e["action"] == "click" && el != nil && el["id"] == "send"
	})
	if click == nil {
		t.Fatal("нет действия click по #send")
	}
	clicks := 0
	for _, e := range events {
		if el, _ := e["element"].(map[string]any); e["kind"] == "action" && e["action"] == "click" && el != nil && el["id"] == "send" {
			clicks++
		}
	}
	if clicks != 1 {
		t.Errorf("клик по #send записан %d раз(а), ждали 1", clicks)
	}
	if findEvent(events, func(e map[string]any) bool { return e["kind"] == "action" && e["action"] == "submit" }) == nil {
		t.Error("нет действия submit")
	}
	change := findEvent(events, func(e map[string]any) bool {
		sel, _ := e["selected"].([]any)
		return e["kind"] == "action" && e["action"] == "change" && len(sel) == 1 && sel[0] == "Отзыв не относится к товару"
	})
	if change == nil {
		t.Error("нет действия change с выбранным основанием")
	}
	if bytes.Contains(files["events.jsonl"], []byte("Покупатель пишет не о нашем товаре\",\"element")) {
		t.Error("значение поля попало в журнал действий")
	}
	if findEvent(events, func(e map[string]any) bool { return e["kind"] == "navigation" && e["main_frame"] == true }) == nil {
		t.Error("нет события navigation")
	}

	summary := string(files["summary.md"])
	clickN := int(click["n"].(float64))
	if !strings.Contains(summary, fmt.Sprintf("после #%d: POST", clickN)) && !strings.Contains(summary, "после #"+fmt.Sprint(clickN+1)+": POST") {
		t.Errorf("в summary.md запрос не связан с кликом #%d:\n%s", clickN, summary)
	}
	for _, want := range []string{"| POST | `127.0.0.1:", "/api/complaint` | 1 | 200×1 |", "?token", "reason", "access_token", "### Шаг 01 «после жалобы»"} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary.md без %q:\n%s", want, summary)
		}
	}
	if t.Failed() {
		t.Logf("summary.md:\n%s", summary)
	}
}

// Новая вкладка, открытая кабинетом, записывается с самого первого запроса: цель ставится на паузу
// до подписки на сеть.
func TestProbeRecordsNewTabFromFirstRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	env := startProbe(ctx, t)
	env.user.click(ctx, "#open")
	env.rec.waitCompleted(t, "/api/whoami")
	zipPath, err := env.rec.Finish()
	if err != nil {
		t.Fatal(err)
	}
	events := parseEvents(t, readZip(t, zipPath)["events.jsonl"])
	first := findEvent(events, func(e map[string]any) bool { return e["kind"] == "target" && e["event"] == "attached" })
	tab := findEvent(events, func(e map[string]any) bool {
		return e["kind"] == "target" && e["event"] == "attached" && e["opener_id"] == first["target_id"]
	})
	if tab == nil {
		t.Fatal("новая вкладка не подключена")
	}
	nav := findEvent(events, func(e map[string]any) bool {
		return e["kind"] == "request" && e["resource_type"] == "Document" && strings.HasSuffix(str(e["url"]), "/popup")
	})
	if nav == nil || nav["target_id"] != tab["target_id"] {
		t.Fatalf("навигация новой вкладки не записана: %v", nav)
	}
	who := findEvent(events, func(e map[string]any) bool {
		return e["kind"] == "request" && strings.HasSuffix(str(e["url"]), "/api/whoami")
	})
	if who == nil || who["target_id"] != tab["target_id"] {
		t.Fatalf("запрос новой вкладки не записан: %v", who)
	}
	resp := findEvent(events, func(e map[string]any) bool { return e["kind"] == "response" && e["id"] == who["id"] })
	if resp == nil || !strings.Contains(str(resp["body"]), `"supplier":"ИП Тест"`) {
		t.Fatalf("ответ новой вкладки: %v", resp)
	}
}

func (r *Recorder) hasCompleted(path string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, it := range r.idx.items {
		if it.kind == itemRequest && strings.Contains(it.req.url, path) && it.req.done {
			return true
		}
	}
	return false
}

func readZip(t *testing.T, path string) map[string][]byte {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	out := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		out[f.Name] = data
	}
	return out
}

func parseEvents(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	var out []map[string]any
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(nil, 64<<20)
	prevSeq := 0.0
	for sc.Scan() {
		var e map[string]any
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("строка events.jsonl не JSON: %v", err)
		}
		if seq := e["seq"].(float64); seq != prevSeq+1 {
			t.Fatalf("seq %v после %v", seq, prevSeq)
		}
		prevSeq = e["seq"].(float64)
		out = append(out, e)
	}
	return out
}

func findEvent(events []map[string]any, f func(map[string]any) bool) map[string]any {
	for _, e := range events {
		if f(e) {
			return e
		}
	}
	return nil
}

func headerAny(h map[string]any, name string) any {
	for k, v := range h {
		if strings.EqualFold(k, name) {
			return v
		}
	}
	return nil
}

func str(v any) string { s, _ := v.(string); return s }

func keys(m map[string][]byte) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
