package probe

import (
	"strings"
	"testing"
)

func TestRedactHeaders(t *testing.T) {
	in := map[string]string{
		"Cookie":          "sid=abc; x-supplier-id-external=0f1e; wbx-validation-key=zzz",
		"Set-Cookie":      "session=s1; Path=/; HttpOnly\nWBTokenV3=t2; Domain=.wildberries.ru",
		"Authorization":   "Bearer abc.def",
		"AuthorizeV3":     "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.sig",
		"X-Access-Token":  "t",
		"X-Session-Id":    "s",
		"X-Client-Secret": "c",
		"X-Signature":     "sg",
		"Wb-Seller-Lk":    "lk",
		"Referer":         "https://seller.wildberries.ru/feedbacks?token=abc&page=2",
		"Content-Type":    "application/json",
		"X-Supplier-Id":   "0f1e", // идентификатор кабинета нужен для проверки кабинета, не секрет
	}
	got := RedactHeaders(in)
	want := map[string]string{
		"Cookie":          "sid=" + Hidden + "; x-supplier-id-external=" + Hidden + "; wbx-validation-key=" + Hidden,
		"Set-Cookie":      "session=" + Hidden + "; Path=/; HttpOnly\nWBTokenV3=" + Hidden + "; Domain=.wildberries.ru",
		"Authorization":   Hidden,
		"AuthorizeV3":     Hidden,
		"X-Access-Token":  Hidden,
		"X-Session-Id":    Hidden,
		"X-Client-Secret": Hidden,
		"X-Signature":     Hidden,
		"Wb-Seller-Lk":    Hidden,
		"Referer":         "https://seller.wildberries.ru/feedbacks?token=" + Hidden + "&page=2",
		"Content-Type":    "application/json",
		"X-Supplier-Id":   "0f1e",
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s: получили %q, ждали %q", k, got[k], w)
		}
	}
	if in["Authorization"] != "Bearer abc.def" {
		t.Error("RedactHeaders изменил исходную карту")
	}
}

func TestRedactHeaderValueJWT(t *testing.T) {
	// Имя заголовка ничего не говорит, но значение — JWT.
	got := RedactHeaders(map[string]string{"X-Foo": "v=eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abcdef"})
	if strings.Contains(got["X-Foo"], "eyJ") {
		t.Errorf("JWT не скрыт: %q", got["X-Foo"])
	}
}

func TestRedactURL(t *testing.T) {
	cases := map[string]string{
		"https://h/p?token=abc":                            "https://h/p?token=" + Hidden,
		"https://h/p?a=1&access_token=abc&b=2":             "https://h/p?a=1&access_token=" + Hidden + "&b=2",
		"https://h/p?Auth=1&sessionId=2&sign=3&secret_k=4": "https://h/p?Auth=" + Hidden + "&sessionId=" + Hidden + "&sign=" + Hidden + "&secret_k=" + Hidden,
		"https://h/p?page=1&sort=desc":                     "https://h/p?page=1&sort=desc",
		"https://h/cb#access_token=abc&state=1":            "https://h/cb#access_token=" + Hidden + "&state=1",
		"https://h/p#section":                              "https://h/p#section",
		"https://user:pass@h/p":                            "https://" + Hidden + "@h/p",
		"https://h/p?redirect=https%3A%2F%2Fx%2F%3Ftoken%3Dabc": "https://h/p?redirect=" +
			"https%3A%2F%2Fx%2F%3Ftoken%3D%3C%D1%81%D0%BA%D1%80%D1%8B%D1%82%D0%BE%3E",
		"https://h/p?%74oken=abc": "https://h/p?%74oken=" + Hidden, // имя в percent-кодировке
		"about:blank":             "about:blank",
		"":                        "",
	}
	for in, want := range cases {
		if got := RedactURL(in); got != want {
			t.Errorf("RedactURL(%q) = %q, ждали %q", in, got, want)
		}
	}
}

func TestRedactBodyJSON(t *testing.T) {
	in := `{"z":1,"token":"abc","nested":{"refreshToken":"r","keep":"v","session":{"id":5}},` +
		`"list":[{"apiKey":"k"},{"n":2.50}],"text":"<b>ok</b>","auth":null,"flag":true}`
	want := `{"z":1,"token":"` + Hidden + `","nested":{"refreshToken":"` + Hidden + `","keep":"v","session":"` + Hidden + `"},` +
		`"list":[{"apiKey":"` + Hidden + `"},{"n":2.50}],"text":"<b>ok</b>","auth":"` + Hidden + `","flag":true}`
	if got := RedactBody(in, "application/json; charset=utf-8"); got != want {
		t.Errorf("получили\n%s\nждали\n%s", got, want)
	}
	// Без content-type, но похоже на JSON.
	if got := RedactBody(`[{"csrf":"x"}]`, ""); got != `[{"csrf":"`+Hidden+`"}]` {
		t.Errorf("массив: %s", got)
	}
	// Секрет внутри строкового значения.
	if got := RedactBody(`{"url":"https://h/?token=abc"}`, "application/json"); strings.Contains(got, "abc") {
		t.Errorf("строка со ссылкой: %s", got)
	}
}

func TestRedactBodyForm(t *testing.T) {
	got := RedactBody("a=1&password=p&csrf_token=c&text=%D0%BF", "application/x-www-form-urlencoded")
	want := "a=1&password=" + Hidden + "&csrf_token=" + Hidden + "&text=%D0%BF"
	if got != want {
		t.Errorf("получили %q, ждали %q", got, want)
	}
}

func TestRedactBodyMultipart(t *testing.T) {
	body := "--XB\r\nContent-Disposition: form-data; name=\"text\"\r\n\r\nтекст жалобы\r\n" +
		"--XB\r\nContent-Disposition: form-data; name=\"token\"\r\n\r\nsecret-value\r\n" +
		"--XB\r\nContent-Disposition: form-data; name=\"file\"; filename=\"a.png\"\r\nContent-Type: image/png\r\n\r\n\x89PNGbinary\r\n" +
		"--XB--\r\n"
	got := RedactBody(body, "multipart/form-data; boundary=XB")
	for _, bad := range []string{"secret-value", "PNGbinary"} {
		if strings.Contains(got, bad) {
			t.Errorf("%q не скрыто:\n%s", bad, got)
		}
	}
	for _, want := range []string{"текст жалобы", `name="token"`, "<файл: 10 байт>", `filename="a.png"`} {
		if !strings.Contains(got, want) {
			t.Errorf("нет %q:\n%s", want, got)
		}
	}
}

func TestRedactBodyText(t *testing.T) {
	in := `<script>window.cfg = {"accessToken": "abc", sessionKey: 'def', page: 1}; var auth_key=ghi;</script>` +
		` Authorization: Bearer 0123456789abcdef`
	got := RedactBody(in, "text/html")
	for _, bad := range []string{"abc", "def", "ghi", "0123456789abcdef"} {
		if strings.Contains(got, bad) {
			t.Errorf("%q не скрыто: %s", bad, got)
		}
	}
	if !strings.Contains(got, "page: 1") {
		t.Errorf("лишнее скрыто: %s", got)
	}
}

func TestIsSecretName(t *testing.T) {
	for _, n := range []string{"token", "X-Auth", "sessionid", "client_secret", "signature", "Cookie", "sid", "_csrf", "passwd", "api-key"} {
		if !IsSecretName(n) {
			t.Errorf("%q должно считаться секретом", n)
		}
	}
	for _, n := range []string{"page", "nmId", "text", "supplierFeedbackValuation", "side", "x-supplier-id"} {
		if IsSecretName(n) {
			t.Errorf("%q не должно считаться секретом", n)
		}
	}
}
