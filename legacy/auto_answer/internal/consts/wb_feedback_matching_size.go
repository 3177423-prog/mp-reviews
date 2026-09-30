package consts

import (
	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

// Соответствие заявленного размера реальному
type WbFeedbackMatchingSizeType string

type OptNilWbFeedbackMatchingSizeType struct {
	Value WbFeedbackMatchingSizeType
	Set   bool
	Null  bool
}

const (
	// `` — для безразмерных товаров.
	WbFeedbackMatchingSizeTypeSPACE WbFeedbackMatchingSizeType = ""
	// `ок` — соответствует размеру.
	WbFeedbackMatchingSizeTypeOK WbFeedbackMatchingSizeType = "ок"
	// `smaller` — маломерит.
	WbFeedbackMatchingSizeTypeSMALLER WbFeedbackMatchingSizeType = "smaller"
	// `bigger` — большемерит.
	WbFeedbackMatchingSizeTypeBIGGER WbFeedbackMatchingSizeType = "bigger"
)

func (WbFeedbackMatchingSizeType) AllValues() []WbFeedbackMatchingSizeType {
	return []WbFeedbackMatchingSizeType{
		WbFeedbackMatchingSizeTypeSPACE,
		WbFeedbackMatchingSizeTypeOK,
		WbFeedbackMatchingSizeTypeSMALLER,
		WbFeedbackMatchingSizeTypeBIGGER,
	}
}

func (s WbFeedbackMatchingSizeType) MarshalText() ([]byte, error) {
	switch s {
	case WbFeedbackMatchingSizeTypeSPACE:
		return []byte(s), nil
	case WbFeedbackMatchingSizeTypeOK:
		return []byte(s), nil
	case WbFeedbackMatchingSizeTypeSMALLER:
		return []byte(s), nil
	case WbFeedbackMatchingSizeTypeBIGGER:
		return []byte(s), nil
	default:
		return nil, errors.Errorf("invalid value: %q", s)
	}
}

func (s *WbFeedbackMatchingSizeType) UnmarshalText(data []byte) error {
	switch WbFeedbackMatchingSizeType(data) {
	case WbFeedbackMatchingSizeTypeSPACE:
		*s = WbFeedbackMatchingSizeTypeSPACE
		return nil
	case WbFeedbackMatchingSizeTypeOK:
		*s = WbFeedbackMatchingSizeTypeOK
		return nil
	case WbFeedbackMatchingSizeTypeSMALLER:
		*s = WbFeedbackMatchingSizeTypeSMALLER
		return nil
	case WbFeedbackMatchingSizeTypeBIGGER:
		*s = WbFeedbackMatchingSizeTypeBIGGER
		return nil
	default:
		return errors.Errorf("invalid value: %q", data)
	}
}

func (s WbFeedbackMatchingSizeType) Validate() error {
	switch s {
	case "":
		return nil
	case "ок":
		return nil
	case "smaller":
		return nil
	case "bigger":
		return nil
	default:
		return errors.Errorf("invalid value: %v", s)
	}
}

func (s WbFeedbackMatchingSizeType) Encode(e *jx.Encoder) {
	e.Str(string(s))
}

func (s *WbFeedbackMatchingSizeType) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode WbFeedbackMatchingSizeType to nil")
	}
	v, err := d.StrBytes()
	if err != nil {
		return err
	}
	// Try to use constant string.
	switch WbFeedbackMatchingSizeType(v) {
	case WbFeedbackMatchingSizeTypeSPACE:
		*s = WbFeedbackMatchingSizeTypeSPACE
	case WbFeedbackMatchingSizeTypeOK:
		*s = WbFeedbackMatchingSizeTypeOK
	case WbFeedbackMatchingSizeTypeSMALLER:
		*s = WbFeedbackMatchingSizeTypeSMALLER
	case WbFeedbackMatchingSizeTypeBIGGER:
		*s = WbFeedbackMatchingSizeTypeBIGGER
	default:
		*s = WbFeedbackMatchingSizeType(v)
	}

	return nil
}

func (s WbFeedbackMatchingSizeType) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *WbFeedbackMatchingSizeType) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}

func (o OptNilWbFeedbackMatchingSizeType) IsSet() bool { return o.Set }

func (o *OptNilWbFeedbackMatchingSizeType) Reset() {
	var v WbFeedbackMatchingSizeType
	o.Value = v
	o.Set = false
	o.Null = false
}

func (o *OptNilWbFeedbackMatchingSizeType) SetTo(v WbFeedbackMatchingSizeType) {
	o.Set = true
	o.Null = false
	o.Value = v
}

func (o OptNilWbFeedbackMatchingSizeType) IsNull() bool { return o.Null }

func (o *OptNilWbFeedbackMatchingSizeType) SetToNull() {
	o.Set = true
	o.Null = true
	var v WbFeedbackMatchingSizeType
	o.Value = v
}

func (o OptNilWbFeedbackMatchingSizeType) Get() (v WbFeedbackMatchingSizeType, ok bool) {
	if o.Null {
		return v, false
	}
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptNilWbFeedbackMatchingSizeType) Or(d WbFeedbackMatchingSizeType) WbFeedbackMatchingSizeType {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptNilWbFeedbackMatchingSizeType) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	if o.Null {
		e.Null()
		return
	}
	e.Str(string(o.Value))
}

func (o *OptNilWbFeedbackMatchingSizeType) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("invalid: unable to decode OptNilWbFeedbackMatchingSizeType to nil")
	}
	if d.Next() == jx.Null {
		if err := d.Null(); err != nil {
			return err
		}

		var v WbFeedbackMatchingSizeType
		o.Value = v
		o.Set = true
		o.Null = true
		return nil
	}
	o.Set = true
	o.Null = false
	if err := o.Value.Decode(d); err != nil {
		return err
	}
	return nil
}

func (s OptNilWbFeedbackMatchingSizeType) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *OptNilWbFeedbackMatchingSizeType) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
