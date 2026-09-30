package domain

import (
	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

// Фотография в отзыве
type WbFeedbackPhotoItem struct {
	FullSize string `json:"fullSize"` // Адрес фотографии полного размера
	MinSize  string `json:"miniSize"` // Адрес фотографии маленького размера
}

// Массив структур фотографий
type WbFeedbackPhotoArray []WbFeedbackPhotoItem

type OptNilWbFeedbackPhotoArray struct {
	Value WbFeedbackPhotoArray
	Set   bool
	Null  bool
}

func (s *WbFeedbackPhotoItem) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode WbFeedbackPhotoItem to nil")
	}

	if err := d.ObjBytes(func(d *jx.Decoder, k []byte) error {
		switch string(k) {
		case "fullSize":
			if err := func() error {
				v, err := d.Str()
				s.FullSize = string(v)
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"fullSize\"")
			}
		case "miniSize":
			if err := func() error {
				v, err := d.Str()
				s.FullSize = string(v)
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"miniSize\"")
			}
		default:
			return d.Skip()
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "decode WbFeedbackPhotoItem")
	}

	return nil
}

func (s *WbFeedbackPhotoItem) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}

func (s *WbFeedbackPhotoArray) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode WbFeedbackPhotoArray to nil")
	}

	if err := d.ObjBytes(func(d *jx.Decoder, k []byte) error {
		wrapped := make(WbFeedbackPhotoArray, 0)
		if err := d.Arr(func(d *jx.Decoder) error {
			var elem WbFeedbackPhotoItem
			if err := elem.Decode(d); err != nil {
				return err
			}
			wrapped = append(wrapped, elem)
			return nil
		}); err != nil {
			return err
		}

		s = &wrapped
		return nil
	}); err != nil {
		return errors.Wrap(err, "decode WbFeedbackPhotoArray")
	}

	return nil
}

func (s *WbFeedbackPhotoArray) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}

func (o OptNilWbFeedbackPhotoArray) IsSet() bool { return o.Set }

func (o *OptNilWbFeedbackPhotoArray) Reset() {
	var v WbFeedbackPhotoArray
	o.Value = v
	o.Set = false
	o.Null = false
}

func (o OptNilWbFeedbackPhotoArray) IsNull() bool { return o.Null }

func (o OptNilWbFeedbackPhotoArray) Or(d WbFeedbackPhotoArray) WbFeedbackPhotoArray {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptNilWbFeedbackPhotoArray) Get() (v WbFeedbackPhotoArray, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o *OptNilWbFeedbackPhotoArray) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("invalid: unable to decode OptNilWbFeedbackPhotoArray to nil")
	}
	if d.Next() == jx.Null {
		if err := d.Null(); err != nil {
			return err
		}

		var v WbFeedbackPhotoArray
		o.Value = v
		o.Set = true
		o.Null = true
		return nil
	}
	o.Set = true
	o.Null = false
	o.Value = make(WbFeedbackPhotoArray, 0)
	if err := d.Arr(func(d *jx.Decoder) error {
		var elem WbFeedbackPhotoItem
		if err := elem.Decode(d); err != nil {
			return err
		}
		o.Value = append(o.Value, elem)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *OptNilWbFeedbackPhotoArray) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
