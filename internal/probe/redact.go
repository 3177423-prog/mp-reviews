package probe

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Hidden — то, чем заменяются значения секретов. Имена заголовков, параметров, cookie и ключей
// JSON сохраняются, чтобы по записи восстанавливалась структура запросов.
const Hidden = "<скрыто>"

// secretWords — подстроки имён, значения которых вычищаются. По SPEC §11 это token, auth, session,
// secret, sign; остальное добавлено консервативно (cookie, CSRF, пароли, API-ключи, JWT).
const secretWords = `token|auth|session|secret|sign|cookie|csrf|xsrf|passw|jwt|api[-_]?key|wb-seller-lk`

var (
	secretNameRe = regexp.MustCompile(`(?i)(` + secretWords + `)`)
	// exactSecretNames — короткие имена, которые нельзя искать подстрокой.
	exactSecretNames = map[string]bool{"sid": true, "ssid": true, "pwd": true}

	// textSecretRe — «имя=значение» / «"имя": "значение"» в произвольном тексте (HTML, JS, text/plain).
	textSecretRe = regexp.MustCompile(`(?i)(["']?[A-Za-z0-9_.\-]*(?:` + secretWords + `)[A-Za-z0-9_.\-]*["']?\s*[:=]\s*)` +
		`("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^\s&,;}\]"'<>]+)`)
	bearerRe = regexp.MustCompile(`(?i)\b(bearer|basic)\s+[A-Za-z0-9._~+/=\-]{6,}`)
	jwtRe    = regexp.MustCompile(`eyJ[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]*`)
)

// IsSecretName — имя заголовка, параметра, cookie или ключа JSON указывает на секрет.
func IsSecretName(name string) bool {
	if exactSecretNames[strings.ToLower(strings.TrimSpace(name))] {
		return true
	}
	return secretNameRe.MatchString(name)
}

// RedactHeaders возвращает копию заголовков с вычищенными секретами. Значения заголовков CDP —
// строки, несколько значений одного заголовка разделены переводом строки.
func RedactHeaders(h map[string]string) map[string]string {
	if h == nil {
		return nil
	}
	out := make(map[string]string, len(h))
	for name, v := range h {
		out[name] = redactHeader(name, v)
	}
	return out
}

func redactHeader(name, v string) string {
	switch strings.ToLower(name) {
	case "cookie":
		return redactCookieHeader(v)
	case "set-cookie":
		lines := strings.Split(v, "\n")
		for i, l := range lines {
			lines[i] = redactSetCookie(l)
		}
		return strings.Join(lines, "\n")
	case "referer", "location", "content-location", "origin", "x-forwarded-for-url":
		return RedactURL(v)
	}
	if IsSecretName(name) {
		return Hidden
	}
	return redactText(v)
}

// redactCookieHeader: "a=1; b=2" → "a=<скрыто>; b=<скрыто>".
func redactCookieHeader(v string) string {
	parts := strings.Split(v, ";")
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		name, _, _ := strings.Cut(p, "=")
		parts[i] = name + "=" + Hidden
	}
	return strings.Join(parts, "; ")
}

// redactSetCookie: "sid=abc; Path=/; HttpOnly" → "sid=<скрыто>; Path=/; HttpOnly".
func redactSetCookie(v string) string {
	first, rest, hasRest := strings.Cut(v, ";")
	name, _, _ := strings.Cut(strings.TrimSpace(first), "=")
	out := name + "=" + Hidden
	if hasRest {
		out += ";" + rest
	}
	return out
}

// RedactURL вычищает значения секретных параметров в query и во фрагменте, логин и пароль
// в адресе и JWT в пути. Порядок и написание остальных параметров сохраняются.
func RedactURL(raw string) string {
	if raw == "" {
		return raw
	}
	s := raw
	frag := ""
	if i := strings.IndexByte(s, '#'); i >= 0 {
		s, frag = s[:i], s[i+1:]
	}
	query := ""
	hasQuery := false
	if i := strings.IndexByte(s, '?'); i >= 0 {
		s, query, hasQuery = s[:i], s[i+1:], true
	}
	// user:pass@host
	if i := strings.Index(s, "://"); i >= 0 {
		rest := s[i+3:]
		hostEnd := strings.IndexByte(rest, '/')
		if hostEnd < 0 {
			hostEnd = len(rest)
		}
		if at := strings.LastIndexByte(rest[:hostEnd], '@'); at >= 0 {
			s = s[:i+3] + Hidden + "@" + rest[at+1:]
		}
	}
	s = jwtRe.ReplaceAllString(s, Hidden)
	if hasQuery {
		s += "?" + redactQuery(query)
	}
	if frag != "" {
		if strings.Contains(frag, "=") {
			frag = redactQuery(frag)
		}
		s += "#" + frag
	} else if strings.HasSuffix(raw, "#") {
		s += "#"
	}
	return s
}

// redactQuery обрабатывает "a=1&token=2" (и тела application/x-www-form-urlencoded).
func redactQuery(q string) string {
	if q == "" {
		return q
	}
	parts := strings.Split(q, "&")
	for i, p := range parts {
		name, val, hasVal := strings.Cut(p, "=")
		dec, err := url.QueryUnescape(name)
		if err != nil {
			dec = name
		}
		switch {
		case IsSecretName(dec):
			parts[i] = name + "=" + Hidden
		case hasVal:
			if uv, err := url.QueryUnescape(val); err == nil && uv != val {
				// Значение может само быть адресом с секретом (redirect_uri=...).
				if red := RedactURL(uv); red != uv {
					parts[i] = name + "=" + url.QueryEscape(red)
					continue
				}
			}
			parts[i] = name + "=" + jwtRe.ReplaceAllString(val, Hidden)
		}
	}
	return strings.Join(parts, "&")
}

// RedactBody вычищает секреты из тела запроса или ответа. Структура сохраняется: в JSON
// заменяются значения ключей с секретными именами, в формах — значения таких полей,
// содержимое файлов в multipart заменяется размером.
func RedactBody(body, contentType string) string {
	if body == "" {
		return body
	}
	mt, params, _ := mime.ParseMediaType(contentType)
	switch {
	case mt == "multipart/form-data" && params["boundary"] != "":
		if out, err := redactMultipart(body, params["boundary"]); err == nil {
			return out
		}
	case mt == "application/x-www-form-urlencoded":
		return redactQuery(body)
	}
	trimmed := strings.TrimSpace(body)
	if strings.Contains(mt, "json") || strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		if out, ok := redactJSON(body); ok {
			return out
		}
	}
	return redactText(body)
}

// redactText — эвристика для неструктурированного текста.
func redactText(s string) string {
	s = jwtRe.ReplaceAllString(s, Hidden)
	s = bearerRe.ReplaceAllString(s, "$1 "+Hidden)
	s = textSecretRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := textSecretRe.FindStringSubmatch(m)
		prefix, val := sub[1], sub[2]
		switch {
		case strings.HasPrefix(val, `"`):
			return prefix + `"` + Hidden + `"`
		case strings.HasPrefix(val, `'`):
			return prefix + `'` + Hidden + `'`
		}
		return prefix + Hidden
	})
	return s
}

// redactJSON переписывает JSON потоково, сохраняя порядок ключей.
func redactJSON(s string) (string, bool) {
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var buf bytes.Buffer
	if err := copyJSONValue(dec, &buf); err != nil {
		return "", false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return "", false
	}
	return buf.String(), true
}

func copyJSONValue(dec *json.Decoder, buf *bytes.Buffer) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			buf.WriteByte('{')
			first := true
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return err
				}
				key, ok := kt.(string)
				if !ok {
					return errors.New("json: ключ не строка")
				}
				if !first {
					buf.WriteByte(',')
				}
				first = false
				writeJSONString(buf, key)
				buf.WriteByte(':')
				if IsSecretName(key) {
					if err := skipJSONValue(dec); err != nil {
						return err
					}
					writeJSONString(buf, Hidden)
					continue
				}
				if err := copyJSONValue(dec, buf); err != nil {
					return err
				}
			}
			if _, err := dec.Token(); err != nil {
				return err
			}
			buf.WriteByte('}')
		case '[':
			buf.WriteByte('[')
			first := true
			for dec.More() {
				if !first {
					buf.WriteByte(',')
				}
				first = false
				if err := copyJSONValue(dec, buf); err != nil {
					return err
				}
			}
			if _, err := dec.Token(); err != nil {
				return err
			}
			buf.WriteByte(']')
		default:
			return errors.New("json: неожиданный разделитель")
		}
	case string:
		writeJSONString(buf, redactText(t))
	case json.Number:
		buf.WriteString(t.String())
	case bool:
		if t {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case nil:
		buf.WriteString("null")
	}
	return nil
}

func skipJSONValue(dec *json.Decoder) error {
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		if d, ok := tok.(json.Delim); ok {
			switch d {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
		}
		if depth == 0 {
			return nil
		}
	}
}

func writeJSONString(buf *bytes.Buffer, s string) {
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	buf.Truncate(buf.Len() - 1) // Encode дописывает перевод строки
}

// redactMultipart пересобирает multipart/form-data: секретные поля и содержимое файлов скрываются.
func redactMultipart(body, boundary string) (string, error) {
	r := multipart.NewReader(strings.NewReader(body), boundary)
	var b strings.Builder
	for {
		p, err := r.NextRawPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		data, err := io.ReadAll(p)
		if err != nil {
			return "", err
		}
		b.WriteString("--" + boundary + "\r\n")
		names := make([]string, 0, len(p.Header))
		for k := range p.Header {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, k := range names {
			for _, v := range p.Header[k] {
				b.WriteString(k + ": " + redactText(v) + "\r\n")
			}
		}
		b.WriteString("\r\n")
		switch {
		case p.FileName() != "":
			b.WriteString("<файл: " + strconv.Itoa(len(data)) + " байт>")
		case IsSecretName(p.FormName()):
			b.WriteString(Hidden)
		default:
			b.WriteString(RedactBody(string(data), p.Header.Get("Content-Type")))
		}
		b.WriteString("\r\n")
	}
	b.WriteString("--" + boundary + "--\r\n")
	return b.String(), nil
}
