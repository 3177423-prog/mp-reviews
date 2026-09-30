package probe

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPathTemplate(t *testing.T) {
	cases := map[string]string{
		"https://seller.wildberries.ru/ns/api/v2/feedbacks/123456?x=1": "seller.wildberries.ru/ns/api/v2/feedbacks/{n}",
		"https://h/a/0f1e2d3c-4b5a-6978-8a9b-0c1d2e3f4a5b/b":           "h/a/{uuid}/b",
		"https://h/a/0123456789abcdef0123/b":                           "h/a/{hex}/b",
		"https://h/r/AbCdEfGh12345678IjKlMn":                           "h/r/{id}",
		"https://h/r/complaints-status":                                "h/r/complaints-status",
		"https://h/":                                                   "h/",
	}
	for in, want := range cases {
		if got := pathTemplate(in); got != want {
			t.Errorf("pathTemplate(%q) = %q, ждали %q", in, got, want)
		}
	}
}

func TestTopLevelKeys(t *testing.T) {
	cases := []struct {
		body, ct string
		want     []string
	}{
		{`{"b":1,"a":{"x":1}}`, "application/json", []string{"a", "b"}},
		{`[{"id":1,"text":"t"}]`, "application/json", []string{"[]id", "[]text"}},
		{`[1,2]`, "application/json", []string{"[]"}},
		{`a=1&b=2`, "application/x-www-form-urlencoded", []string{"a", "b"}},
		{"--B\r\nContent-Disposition: form-data; name=\"f\"\r\n\r\nv\r\n--B--", "multipart/form-data; boundary=B", []string{"f"}},
		{`<html>`, "text/html", nil},
		{``, "application/json", nil},
	}
	for _, c := range cases {
		if got := topLevelKeys(c.body, c.ct); !reflect.DeepEqual(got, c.want) {
			t.Errorf("topLevelKeys(%q) = %v, ждали %v", c.body, got, c.want)
		}
	}
	if got := queryKeys("https://h/p?b=1&a=2&b=3"); !reflect.DeepEqual(got, []string{"?b", "?a"}) {
		t.Errorf("queryKeys: %v", got)
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Открыл список отзывов":  "otkryl-spisok-otzyvov",
		"жалоба: диалог / шаг 2": "zhaloba-dialog-shag-2",
		`<>:"/\|?*`:              "step",
		"Ozon  ответ":            "ozon-otvet",
		"объём":                  "obem",
	}
	for in, want := range cases {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, ждали %q", in, got, want)
		}
	}
	if n := len(slug(strings.Repeat("щ", 100))); n > 60 {
		t.Errorf("длина slug %d", n)
	}
}

func TestLinksAndRender(t *testing.T) {
	t0 := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	at := func(ms int) time.Time { return t0.Add(time.Duration(ms) * time.Millisecond) }
	bg := &reqItem{id: "r1", method: "GET", url: "https://h/api/poll", resType: "XHR", status: 200, done: true}
	sub := &reqItem{id: "r2", method: "POST", url: "https://h/api/complaint/123?token=" + Hidden, resType: "Fetch",
		status: 200, done: true, reqKeys: []string{"?token", "reason"}, respKeys: []string{"id"}}
	late := &reqItem{id: "r3", method: "GET", url: "https://h/api/poll", resType: "XHR", status: 498, done: true}
	x := index{profile: "wb-ms", start: t0, end: at(60000)}
	x.add(item{ts: at(100), kind: itemRequest, req: bg})
	x.add(item{ts: at(1000), kind: itemMark, n: 1, desc: "открыл отзыв", files: []string{"marks/01-a.html", ""}})
	x.add(item{ts: at(2000), kind: itemAction, n: 1, desc: "клик button «Пожаловаться»"})
	x.add(item{ts: at(2150), kind: itemRequest, req: sub})
	x.add(item{ts: at(30000), kind: itemRequest, req: late})

	l := links(x.sorted(), 10*time.Second)
	if l["r1"].action != 0 || l["r1"].mark != 0 {
		t.Errorf("r1: %+v", l["r1"])
	}
	if l["r2"].action != 1 || l["r2"].mark != 1 || l["r2"].afterDelta != 150*time.Millisecond {
		t.Errorf("r2: %+v", l["r2"])
	}
	if l["r3"].action != 0 || l["r3"].mark != 1 {
		t.Errorf("r3 вне окна действия: %+v", l["r3"])
	}

	md := x.render(10 * time.Second)
	for _, want := range []string{
		"| GET | `h/api/poll` | 2 | 200×1, 498×1 | — | — | — |",
		"| POST | `h/api/complaint/{n}` | 1 | 200×1 | ?token, reason | id | #1 |",
		"### Шаг 01 «открыл отзыв»",
		"+150 мс после #1: POST `h/api/complaint/123?token=" + Hidden + "` → 200 [r2]",
		"запрос: GET `h/api/poll` → 498 [r3]",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("нет %q в\n%s", want, md)
		}
	}
}

func TestNewDoesNotOverwritePreviousRecording(t *testing.T) {
	dir := t.TempDir()
	now := func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, time.Local) }
	r1, err := New(nil, Options{Profile: "wb-ms", OutDir: dir, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := New(nil, Options{Profile: "wb-ms", OutDir: dir, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if r1.WorkDir() == r2.WorkDir() {
		t.Fatalf("одинаковый рабочий каталог: %s", r1.WorkDir())
	}
	for _, r := range []*Recorder{r1, r2} {
		if _, err := r.Finish(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "probe-wb-ms-20260930-120000-2.zip")); err != nil {
		t.Errorf("второй zip: %v", err)
	}
}
