package probe

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type itemKind int

const (
	itemRequest itemKind = iota
	itemAction
	itemMark
	itemNav
	itemTab
)

type reqItem struct {
	id       string
	method   string
	url      string
	resType  string
	status   int
	err      string
	reqKeys  []string
	respKeys []string
	done     bool
	redirect bool
}

type item struct {
	ts      time.Time
	kind    itemKind
	n       int
	desc    string
	url     string
	sameDoc bool
	files   []string
	req     *reqItem
}

// index — лёгкая сводка записи в памяти (без тел), из неё строится summary.md.
type index struct {
	profile string
	start   time.Time
	end     time.Time
	items   []item
}

func (x *index) add(it item) { x.items = append(x.items, it) }

func (x *index) sorted() []item {
	out := make([]item, len(x.items))
	copy(out, x.items)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ts.Before(out[j].ts) })
	return out
}

// link — к какому действию и какой отметке шага относится каждый запрос.
type link struct {
	action     int // номер действия, 0 — нет
	actionTS   time.Time
	mark       int // номер отметки, 0 — до первой
	afterDelta time.Duration
}

// links связывает запросы с последним действием пользователя, если запрос начался не позже
// window после него, и с последней отметкой шага.
func links(items []item, window time.Duration) map[string]link {
	out := make(map[string]link)
	lastAction, lastMark := 0, 0
	var lastActionTS time.Time
	for _, it := range items {
		switch it.kind {
		case itemAction:
			lastAction, lastActionTS = it.n, it.ts
		case itemMark:
			lastMark = it.n
		case itemRequest:
			l := link{mark: lastMark}
			if lastAction != 0 {
				if d := it.ts.Sub(lastActionTS); d >= 0 && d <= window {
					l.action, l.actionTS, l.afterDelta = lastAction, lastActionTS, d
				}
			}
			out[it.req.id] = l
		}
	}
	return out
}

type endpoint struct {
	method   string
	path     string
	count    int
	statuses map[string]int
	reqKeys  map[string]bool
	respKeys map[string]bool
	actions  map[int]bool
}

func (x *index) render(window time.Duration) string {
	items := x.sorted()
	lk := links(items, window)

	var b strings.Builder
	fmt.Fprintf(&b, "# Запись пробника кабинета — профиль `%s`\n\n", x.profile)
	fmt.Fprintf(&b, "Начало: %s, конец: %s.\n\n", x.start.Format("2006-01-02 15:04:05"), x.end.Format("2006-01-02 15:04:05"))
	b.WriteString("> Запись содержит тела запросов и ответов кабинета. Значения Cookie, Set-Cookie, Authorization\n" +
		"> и полей с token/auth/session/secret/sign (и похожих) заменены на `" + Hidden + "`,\n" +
		"> но персональные данные покупателей в телах остаются. Файл никуда не публиковать.\n\n")

	var nReq, nDoc, nAct, nMark int
	eps := map[string]*endpoint{}
	var order []string
	for _, it := range items {
		switch it.kind {
		case itemAction:
			nAct++
		case itemMark:
			nMark++
		case itemRequest:
			if it.req.resType == "Document" {
				nDoc++
			} else {
				nReq++
			}
			k := it.req.method + " " + pathTemplate(it.req.url)
			ep := eps[k]
			if ep == nil {
				ep = &endpoint{method: it.req.method, path: pathTemplate(it.req.url), statuses: map[string]int{},
					reqKeys: map[string]bool{}, respKeys: map[string]bool{}, actions: map[int]bool{}}
				eps[k] = ep
				order = append(order, k)
			}
			ep.count++
			ep.statuses[statusLabel(it.req)]++
			for _, k := range it.req.reqKeys {
				ep.reqKeys[k] = true
			}
			for _, k := range it.req.respKeys {
				ep.respKeys[k] = true
			}
			if l := lk[it.req.id]; l.action != 0 {
				ep.actions[l.action] = true
			}
		}
	}
	fmt.Fprintf(&b, "Запросов XHR/fetch: %d, навигаций: %d, действий пользователя: %d, отметок шагов: %d.\n\n", nReq, nDoc, nAct, nMark)

	b.WriteString("## Запросы\n\n")
	b.WriteString("Путь-шаблон: числа → `{n}`, UUID → `{uuid}`, длинные hex и идентификаторы → `{hex}` / `{id}`. " +
		"Ключи запроса: `?имя` — параметры адреса, остальные — верхний уровень тела. " +
		"«Действия» — номера действий из хронологии, за которыми последовал запрос.\n\n")
	b.WriteString("| Метод | Хост + путь | Число | Статусы | Ключи запроса | Ключи ответа | Действия |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	sort.SliceStable(order, func(i, j int) bool { return order[i] < order[j] })
	for _, k := range order {
		ep := eps[k]
		fmt.Fprintf(&b, "| %s | `%s` | %d | %s | %s | %s | %s |\n", ep.method, mdCell(ep.path), ep.count,
			mdCell(joinCounts(ep.statuses)), mdCell(joinKeys(ep.reqKeys)), mdCell(joinKeys(ep.respKeys)), joinInts(ep.actions))
	}

	b.WriteString("\n## Хронология\n\n")
	b.WriteString("Действия пользователя, отметки шагов, переходы и запросы по времени. Запрос, начавшийся не позже " +
		strconv.Itoa(int(window/time.Second)) + " с после действия, показан под этим действием.\n\n")
	b.WriteString("### До первой отметки\n\n")
	for _, it := range items {
		t := it.ts.Format("15:04:05.000")
		switch it.kind {
		case itemMark:
			fmt.Fprintf(&b, "\n### Шаг %02d «%s» — %s\n\n", it.n, mdText(it.desc), t)
			var files []string
			for _, f := range it.files {
				if f != "" {
					files = append(files, "`"+f+"`")
				}
			}
			if len(files) > 0 {
				fmt.Fprintf(&b, "Снимок: %s\n\n", strings.Join(files, ", "))
			}
		case itemAction:
			fmt.Fprintf(&b, "- %s **действие #%d**: %s\n", t, it.n, mdText(it.desc))
		case itemNav:
			kind := "переход"
			if it.sameDoc {
				kind = "переход внутри страницы"
			}
			fmt.Fprintf(&b, "- %s %s: `%s`\n", t, kind, mdCode(it.url))
		case itemTab:
			fmt.Fprintf(&b, "- %s новая вкладка: `%s`\n", t, mdCode(it.url))
		case itemRequest:
			l := lk[it.req.id]
			line := fmt.Sprintf("%s `%s` → %s", it.req.method, mdCode(hostPath(it.req.url)), statusLabel(it.req))
			if it.req.resType == "Document" {
				line += " (навигация)"
			}
			if l.action != 0 {
				fmt.Fprintf(&b, "  - +%d мс после #%d: %s [%s]\n", l.afterDelta.Milliseconds(), l.action, line, it.req.id)
			} else {
				fmt.Fprintf(&b, "- %s запрос: %s [%s]\n", t, line, it.req.id)
			}
		}
	}
	b.WriteString("\nИдентификаторы `[rN]` — поле `id` событий `request` и `response` в `events.jsonl`.\n")
	return b.String()
}

func statusLabel(r *reqItem) string {
	switch {
	case r.status != 0 && r.redirect:
		return strconv.Itoa(r.status) + " (перенаправление)"
	case r.status != 0 && r.err != "":
		return strconv.Itoa(r.status) + " (" + r.err + ")"
	case r.status != 0:
		return strconv.Itoa(r.status)
	case r.err != "":
		return "ошибка: " + r.err
	case !r.done:
		return "нет ответа"
	}
	return "—"
}

func joinCounts(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "×" + strconv.Itoa(m[k])
	}
	return strings.Join(parts, ", ")
}

func joinKeys(m map[string]bool) string {
	if len(m) == 0 {
		return "—"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	const maxKeys = 30
	more := ""
	if len(keys) > maxKeys {
		more = fmt.Sprintf(" … (ещё %d)", len(keys)-maxKeys)
		keys = keys[:maxKeys]
	}
	return strings.Join(keys, ", ") + more
}

func joinInts(m map[int]bool) string {
	if len(m) == 0 {
		return "—"
	}
	ns := make([]int, 0, len(m))
	for n := range m {
		ns = append(ns, n)
	}
	sort.Ints(ns)
	parts := make([]string, len(ns))
	for i, n := range ns {
		parts[i] = "#" + strconv.Itoa(n)
	}
	return strings.Join(parts, ", ")
}

func mdCell(s string) string {
	return strings.NewReplacer("|", `\|`, "\n", " ").Replace(s)
}

func mdCode(s string) string { return strings.ReplaceAll(mdCell(s), "`", "'") }

func mdText(s string) string {
	return strings.NewReplacer("\n", " ", "<", "&lt;", ">", "&gt;").Replace(s)
}

func hostPath(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	s := u.Host + u.EscapedPath()
	if u.RawQuery != "" {
		s += "?" + u.RawQuery
	}
	const maxLen = 200
	if len(s) > maxLen {
		s, _ = truncateUTF8(s, maxLen)
		s += "…"
	}
	return s
}

var (
	uuidRe    = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	hexRe     = regexp.MustCompile(`^[0-9a-fA-F]{16,}$`)
	numRe     = regexp.MustCompile(`^[0-9]+$`)
	idLikeRe  = regexp.MustCompile(`^[A-Za-z0-9_\-]{20,}$`)
	hasDigits = regexp.MustCompile(`[0-9]`)
)

// pathTemplate — хост и путь с подставленными вместо идентификаторов шаблонами.
func pathTemplate(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	segs := strings.Split(u.EscapedPath(), "/")
	for i, s := range segs {
		switch {
		case s == "":
		case numRe.MatchString(s):
			segs[i] = "{n}"
		case uuidRe.MatchString(s):
			segs[i] = "{uuid}"
		case hexRe.MatchString(s):
			segs[i] = "{hex}"
		case idLikeRe.MatchString(s) && hasDigits.MatchString(s):
			segs[i] = "{id}"
		}
	}
	host := u.Host
	if host == "" {
		host = u.Scheme + ":"
	}
	return host + strings.Join(segs, "/")
}

// queryKeys — имена параметров адреса в виде "?имя".
func queryKeys(raw string) []string {
	u, err := url.Parse(raw)
	if err != nil || u.RawQuery == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, p := range strings.Split(u.RawQuery, "&") {
		name, _, _ := strings.Cut(p, "=")
		if dec, err := url.QueryUnescape(name); err == nil {
			name = dec
		}
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, "?"+name)
		}
	}
	return out
}

// topLevelKeys — ключи верхнего уровня тела: объекта JSON, первого объекта массива JSON (с префиксом "[]"),
// полей формы.
func topLevelKeys(body, contentType string) []string {
	if body == "" {
		return nil
	}
	mt, _, _ := mime.ParseMediaType(contentType)
	if mt == "application/x-www-form-urlencoded" {
		keys := queryKeys("x:?" + body)
		for i, k := range keys {
			keys[i] = strings.TrimPrefix(k, "?")
		}
		return keys
	}
	if mt == "multipart/form-data" {
		var keys []string
		for _, m := range multipartNameRe.FindAllStringSubmatch(body, -1) {
			keys = append(keys, m[1])
		}
		return keys
	}
	var v any
	dec := json.NewDecoder(strings.NewReader(body))
	if dec.Decode(&v) != nil {
		return nil
	}
	switch t := v.(type) {
	case map[string]any:
		return mapKeys(t, "")
	case []any:
		if len(t) > 0 {
			if m, ok := t[0].(map[string]any); ok {
				return mapKeys(m, "[]")
			}
		}
		return []string{"[]"}
	}
	return nil
}

var multipartNameRe = regexp.MustCompile(`Content-Disposition: form-data; name="([^"]*)"`)

func mapKeys(m map[string]any, prefix string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, prefix+k)
	}
	sort.Strings(out)
	return out
}

// describeAction — одна строка про действие для хронологии.
func describeAction(a evAction) string {
	verb := map[string]string{"click": "клик", "input": "ввод", "change": "изменение", "submit": "отправка формы"}[a.Action]
	if verb == "" {
		verb = a.Action
	}
	parts := []string{verb}
	if e := a.Element; e != nil {
		el := e.Tag
		if e.ID != "" {
			el += "#" + e.ID
		}
		if e.Name != "" {
			el += "[name=" + e.Name + "]"
		}
		if e.Role != "" {
			el += "[role=" + e.Role + "]"
		}
		parts = append(parts, el)
		switch {
		case e.Text != "":
			parts = append(parts, "«"+e.Text+"»")
		case e.Label != "":
			parts = append(parts, "«"+e.Label+"»")
		}
	}
	if len(a.Selected) > 0 {
		parts = append(parts, "выбрано: «"+strings.Join(a.Selected, "», «")+"»")
	}
	if a.Checked != nil {
		parts = append(parts, "отмечено: "+strconv.FormatBool(*a.Checked))
	}
	if a.ValueLen != nil {
		parts = append(parts, "длина: "+strconv.Itoa(*a.ValueLen))
	}
	if a.Files != nil {
		parts = append(parts, "файлов: "+strconv.Itoa(*a.Files))
	}
	if a.Form != nil && a.Form.Method != "" {
		parts = append(parts, a.Form.Method)
	}
	return strings.Join(parts, " ")
}

// zipDir упаковывает каталог в zip (пишет во временный файл и переименовывает).
func zipDir(dir, zipPath string) error {
	tmp := zipPath + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("probe: zip: %w", err)
	}
	zw := zip.NewWriter(f)
	walkErr := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: filepath.ToSlash(rel), Method: zip.Deflate, Modified: time.Now()})
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = src.Close() }()
		_, err = io.Copy(w, src)
		return err
	})
	closeErr := zw.Close()
	fileErr := f.Close()
	for _, e := range []error{walkErr, closeErr, fileErr} {
		if e != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("probe: zip: %w", e)
		}
	}
	if err := os.Rename(tmp, zipPath); err != nil {
		return fmt.Errorf("probe: zip: %w", err)
	}
	return nil
}
