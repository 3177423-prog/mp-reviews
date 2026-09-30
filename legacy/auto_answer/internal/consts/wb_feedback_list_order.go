package consts

import (
	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

// Сортировка отзывов по дате
type WbOrderFeedbackListType string

type OptWbOrderFeedbackListType struct {
	Value WbOrderFeedbackListType
	Set   bool
}

const (
	// `dateAsc` — по возрастанию даты.
	WbOrderFeedbackListTypeDATEASC WbOrderFeedbackListType = "dateAsc"
	// `dateDesc` — по убыванию даты.
	WbOrderFeedbackListTypeDATEDESC WbOrderFeedbackListType = "dateDesc"
)

func (WbOrderFeedbackListType) AllValues() []WbOrderFeedbackListType {
	return []WbOrderFeedbackListType{
		WbOrderFeedbackListTypeDATEASC,
		WbOrderFeedbackListTypeDATEDESC,
	}
}

func (s WbOrderFeedbackListType) MarshalText() ([]byte, error) {
	switch s {
	case WbOrderFeedbackListTypeDATEASC:
		return []byte(s), nil
	case WbOrderFeedbackListTypeDATEDESC:
		return []byte(s), nil
	default:
		return nil, errors.Errorf("invalid value: %q", s)
	}
}

func (s *WbOrderFeedbackListType) UnmarshalText(data []byte) error {
	switch WbOrderFeedbackListType(data) {
	case WbOrderFeedbackListTypeDATEASC:
		*s = WbOrderFeedbackListTypeDATEASC
		return nil
	case WbOrderFeedbackListTypeDATEDESC:
		*s = WbOrderFeedbackListTypeDATEDESC
		return nil
	default:
		return errors.Errorf("invalid value: %q", data)
	}
}

func (s WbOrderFeedbackListType) Validate() error {
	switch s {
	case "dateAsc":
		return nil
	case "dateDesc":
		return nil
	default:
		return errors.Errorf("invalid value: %v", s)
	}
}

func (s WbOrderFeedbackListType) Encode(e *jx.Encoder) {
	e.Str(string(s))
}

func (s *WbOrderFeedbackListType) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode WbOrderFeedbackListType to nil")
	}
	v, err := d.StrBytes()
	if err != nil {
		return err
	}
	// Try to use constant string.
	switch WbOrderFeedbackListType(v) {
	case WbOrderFeedbackListTypeDATEASC:
		*s = WbOrderFeedbackListTypeDATEASC
	case WbOrderFeedbackListTypeDATEDESC:
		*s = WbOrderFeedbackListTypeDATEDESC
	default:
		*s = WbOrderFeedbackListType(v)
	}

	return nil
}

func (s WbOrderFeedbackListType) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *WbOrderFeedbackListType) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}

func (o OptWbOrderFeedbackListType) IsSet() bool { return o.Set }

func (o *OptWbOrderFeedbackListType) Reset() {
	var v WbOrderFeedbackListType
	o.Value = v
	o.Set = false
}

func (o OptWbOrderFeedbackListType) Get() (v WbOrderFeedbackListType, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptWbOrderFeedbackListType) Or(d WbOrderFeedbackListType) WbOrderFeedbackListType {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptWbOrderFeedbackListType) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	e.Str(string(o.Value))
}

func (o *OptWbOrderFeedbackListType) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("invalid: unable to decode OptWbOrderFeedbackListType to nil")
	}
	o.Set = true
	if err := o.Value.Decode(d); err != nil {
		return err
	}
	return nil
}

func (s OptWbOrderFeedbackListType) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *OptWbOrderFeedbackListType) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
