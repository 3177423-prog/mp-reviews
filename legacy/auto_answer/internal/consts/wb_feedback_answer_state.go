package consts

import (
	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

// Статус отзыва
type WbFeedbackAnswerStateType string

const (
	// `none` — новый.
	WbFeedbackAnswerStateTypeNONE WbFeedbackAnswerStateType = "none"
	// `wbRu` — отображается на сайте.
	WbFeedbackAnswerStateTypeWBRU WbFeedbackAnswerStateType = "wbRu"
	// `reviewRequired` — ответ проходит проверку.
	WbFeedbackAnswerStateTypeREVIEWREQUIRED WbFeedbackAnswerStateType = "reviewRequired"
	// `rejected` — ответ отклонён.
	WbFeedbackAnswerStateTypeREJECTED WbFeedbackAnswerStateType = "rejected"
)

func (WbFeedbackAnswerStateType) AllValues() []WbFeedbackAnswerStateType {
	return []WbFeedbackAnswerStateType{
		WbFeedbackAnswerStateTypeNONE,
		WbFeedbackAnswerStateTypeWBRU,
		WbFeedbackAnswerStateTypeREVIEWREQUIRED,
		WbFeedbackAnswerStateTypeREJECTED,
	}
}

func (s WbFeedbackAnswerStateType) MarshalText() ([]byte, error) {
	switch s {
	case WbFeedbackAnswerStateTypeNONE:
		return []byte(s), nil
	case WbFeedbackAnswerStateTypeWBRU:
		return []byte(s), nil
	case WbFeedbackAnswerStateTypeREVIEWREQUIRED:
		return []byte(s), nil
	case WbFeedbackAnswerStateTypeREJECTED:
		return []byte(s), nil
	default:
		return nil, errors.Errorf("invalid value: %q", s)
	}
}

func (s *WbFeedbackAnswerStateType) UnmarshalText(data []byte) error {
	switch WbFeedbackAnswerStateType(data) {
	case WbFeedbackAnswerStateTypeNONE:
		*s = WbFeedbackAnswerStateTypeNONE
		return nil
	case WbFeedbackAnswerStateTypeWBRU:
		*s = WbFeedbackAnswerStateTypeWBRU
		return nil
	case WbFeedbackAnswerStateTypeREVIEWREQUIRED:
		*s = WbFeedbackAnswerStateTypeREVIEWREQUIRED
		return nil
	case WbFeedbackAnswerStateTypeREJECTED:
		*s = WbFeedbackAnswerStateTypeREJECTED
		return nil
	default:
		return errors.Errorf("invalid value: %q", data)
	}
}

func (s WbFeedbackAnswerStateType) Validate() error {
	switch s {
	case "none":
		return nil
	case "wbRu":
		return nil
	case "reviewRequired":
		return nil
	case "rejected":
		return nil
	default:
		return errors.Errorf("invalid value: %v", s)
	}
}

func (s WbFeedbackAnswerStateType) Encode(e *jx.Encoder) {
	e.Str(string(s))
}

func (s *WbFeedbackAnswerStateType) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode WbFeedbackAnswerStateType to nil")
	}
	v, err := d.StrBytes()
	if err != nil {
		return err
	}
	// Try to use constant string.
	switch WbFeedbackAnswerStateType(v) {
	case WbFeedbackAnswerStateTypeNONE:
		*s = WbFeedbackAnswerStateTypeNONE
	case WbFeedbackAnswerStateTypeWBRU:
		*s = WbFeedbackAnswerStateTypeWBRU
	case WbFeedbackAnswerStateTypeREVIEWREQUIRED:
		*s = WbFeedbackAnswerStateTypeREVIEWREQUIRED
	case WbFeedbackAnswerStateTypeREJECTED:
		*s = WbFeedbackAnswerStateTypeREJECTED
	default:
		*s = WbFeedbackAnswerStateType(v)
	}

	return nil
}

func (s WbFeedbackAnswerStateType) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *WbFeedbackAnswerStateType) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
