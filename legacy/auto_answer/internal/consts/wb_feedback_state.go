package consts

import (
	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

// Статус отзыва
type WbFeedbackStateType string

const (
	// `none` — не обработан (новый).
	WbFeedbackStateTypeNONE WbFeedbackStateType = "none"
	// `wbRu` — обработан.
	WbFeedbackStateTypeWBRU WbFeedbackStateType = "wbRu"
)

func (WbFeedbackStateType) AllValues() []WbFeedbackStateType {
	return []WbFeedbackStateType{
		WbFeedbackStateTypeNONE,
		WbFeedbackStateTypeWBRU,
	}
}

func (s WbFeedbackStateType) MarshalText() ([]byte, error) {
	switch s {
	case WbFeedbackStateTypeNONE:
		return []byte(s), nil
	case WbFeedbackStateTypeWBRU:
		return []byte(s), nil
	default:
		return nil, errors.Errorf("invalid value: %q", s)
	}
}

func (s *WbFeedbackStateType) UnmarshalText(data []byte) error {
	switch WbFeedbackStateType(data) {
	case WbFeedbackStateTypeNONE:
		*s = WbFeedbackStateTypeNONE
		return nil
	case WbFeedbackStateTypeWBRU:
		*s = WbFeedbackStateTypeWBRU
		return nil
	default:
		return errors.Errorf("invalid value: %q", data)
	}
}

func (s WbFeedbackStateType) Validate() error {
	switch s {
	case "none":
		return nil
	case "wbRu":
		return nil
	default:
		return errors.Errorf("invalid value: %v", s)
	}
}

func (s WbFeedbackStateType) Encode(e *jx.Encoder) {
	e.Str(string(s))
}

func (s *WbFeedbackStateType) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode WbFeedbackStateType to nil")
	}
	v, err := d.StrBytes()
	if err != nil {
		return err
	}
	// Try to use constant string.
	switch WbFeedbackStateType(v) {
	case WbFeedbackStateTypeNONE:
		*s = WbFeedbackStateTypeNONE
	case WbFeedbackStateTypeWBRU:
		*s = WbFeedbackStateTypeWBRU
	default:
		*s = WbFeedbackStateType(v)
	}

	return nil
}

func (s WbFeedbackStateType) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *WbFeedbackStateType) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
