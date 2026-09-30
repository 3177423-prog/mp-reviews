package domain

import (
	"github.com/go-faster/errors"
	"github.com/go-faster/jx"

	"auto_answer/internal/consts"
)

// Структура ответа
type WbFeedbackAnswer struct {
	Text     OptString                        `json:"text"`     // Текст ответа
	State    consts.WbFeedbackAnswerStateType `json:"state"`    // Статус ответа
	Editable OptBool                          `json:"editable"` // Признак возможности редактировать ответ
}

type OptNilWbFeedbackAnswer struct {
	Value WbFeedbackAnswer
	Set   bool
	Null  bool
}

func (s *WbFeedbackAnswer) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode WbFeedbackAnswer to nil")
	}

	if err := d.ObjBytes(func(d *jx.Decoder, k []byte) error {
		switch string(k) {
		case "text":
			if err := func() error {
				s.Text.Reset()
				if err := s.Text.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"text\"")
			}
		case "state":
			if err := func() error {
				if err := s.State.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"state\"")
			}
		case "editable":
			if err := func() error {
				s.Editable.Reset()
				if err := s.Editable.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"editable\"")
			}
		default:
			return d.Skip()
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "decode WbFeedbackAnswer")
	}

	return nil
}

func (s *WbFeedbackAnswer) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}

func (o OptNilWbFeedbackAnswer) IsSet() bool { return o.Set }

func (o *OptNilWbFeedbackAnswer) Reset() {
	var v WbFeedbackAnswer
	o.Value = v
	o.Set = false
	o.Null = false
}

func (o *OptNilWbFeedbackAnswer) SetTo(v WbFeedbackAnswer) {
	o.Set = true
	o.Null = false
	o.Value = v
}

func (o OptNilWbFeedbackAnswer) IsNull() bool { return o.Null }

func (o *OptNilWbFeedbackAnswer) SetToNull() {
	o.Set = true
	o.Null = true
	var v WbFeedbackAnswer
	o.Value = v
}

func (o OptNilWbFeedbackAnswer) Get() (v WbFeedbackAnswer, ok bool) {
	if o.Null {
		return v, false
	}
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptNilWbFeedbackAnswer) Or(d WbFeedbackAnswer) WbFeedbackAnswer {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptNilWbFeedbackAnswer) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	if o.Null {
		e.Null()
		return
	}
	o.Encode(e)
}

func (o *OptNilWbFeedbackAnswer) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("недействительно: невозможно декодировать OptNilWbFeedbackAnswer в nil")
	}
	if d.Next() == jx.Null {
		if err := d.Null(); err != nil {
			return err
		}

		var v WbFeedbackAnswer
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

func (o OptNilWbFeedbackAnswer) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	o.Encode(&e)
	return e.Bytes(), nil
}

func (o *OptNilWbFeedbackAnswer) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return o.Decode(d)
}
