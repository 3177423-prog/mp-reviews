package domain

import (
	"strconv"
	"strings"
	"time"

	"github.com/go-faster/jx"
	"github.com/ogen-go/ogen/ogenregex"
)

const (
	dateLayout = "2006-01-02"
	// timeLayout = "15:04:05"
)

var regexMap = map[string]ogenregex.Regexp{
	"uuid": ogenregex.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"),
}

// InvalidContentTypeError сообщает, что декодер получил неожиданный тип контента
type InvalidContentTypeError struct {
	ContentType string
}

// InvalidContentTypeError реализует ошибку
func (e *InvalidContentTypeError) Error() string {
	b := strings.Builder{}
	b.WriteString("unexpected Content-Type: ")
	b.WriteString(e.ContentType)

	return b.String()
}

// InvalidContentType создает новый InvalidContentTypeError
func InvalidContentType(contentType string) error {
	return &InvalidContentTypeError{
		ContentType: contentType,
	}
}

// MinLengthError сообщает, что длина меньше минимума
type MinLengthError struct {
	Len       int
	MinLength int
}

// MinLengthError реализует ошибку
func (e *MinLengthError) Error() string {
	b := strings.Builder{}
	b.WriteString("длина ")
	b.WriteString(strconv.Itoa(e.Len))
	b.WriteString(" меньше минимума ")
	b.WriteString(strconv.Itoa(e.MinLength))

	return b.String()
}

// MaxLengthError сообщает, что длина больше максимума
type MaxLengthError struct {
	Len       int
	MaxLength int
}

// MaxLengthError реализует ошибку
func (e *MaxLengthError) Error() string {
	b := strings.Builder{}
	b.WriteString("длина ")
	b.WriteString(strconv.Itoa(e.Len))
	b.WriteString(" больше минимума ")
	b.WriteString(strconv.Itoa(e.MaxLength))

	return b.String()
}

// EncodeTimeFormat кодирует дату, время и дату-время в JSON, используя пользовательский формат
func EncodeTimeFormat(e *jx.Encoder, v time.Time, layout string) {
	const stackThreshold = 64

	var buf []byte
	if len(layout) > stackThreshold {
		buf = make([]byte, len(layout))
	} else {
		// Выделяем buf в стеке, если можем.
		buf = make([]byte, stackThreshold)
	}

	buf = v.AppendFormat(buf[:0], layout)
	e.ByteStr(buf)
}

// DecodeTimeFormat декодирует дату, время и дату-время из JSON, используя пользовательский формат
func DecodeTimeFormat(d *jx.Decoder, layout string) (v time.Time, err error) {
	s, err := d.Str()
	if err != nil {
		return v, err
	}
	return time.Parse(layout, s)
}

// EncodeDateTime кодирует date-time в json
func EncodeDateTime(e *jx.Encoder, v time.Time) {
	EncodeTimeFormat(e, v, time.RFC3339)
}

// DecodeDateTime декодирует date-time из json
func DecodeDateTime(d *jx.Decoder) (v time.Time, err error) {
	return DecodeTimeFormat(d, time.RFC3339)
}

// DecodeDate декодирует date из json
func DecodeDate(d *jx.Decoder) (v time.Time, err error) {
	return DecodeTimeFormat(d, dateLayout)
}

// EncodeDate кодирует date в json
func EncodeDate(e *jx.Encoder, v time.Time) {
	EncodeTimeFormat(e, v, dateLayout)
}
