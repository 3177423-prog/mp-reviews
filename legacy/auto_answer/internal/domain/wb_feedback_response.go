package domain

import (
	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

// Список отзывов на Wildberries
type WbFeedbackResponse struct {
	Data             WbFeedbacksData   `json:"data"`             // Информация об отзывах
	Error            OptBool           `json:"error"`            // Есть ли ошибка
	ErrorText        OptNilString      `json:"errorText"`        // Описание ошибки
	AdditionalErrors OptNilStringArray `json:"additionalErrors"` // Дополнительные ошибки
}

func (s *WbFeedbackResponse) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode WbFeedbackResponse to nil")
	}

	if err := d.ObjBytes(func(d *jx.Decoder, k []byte) error {
		switch string(k) {
		case "data":
			if err := func() error {
				if err := s.Data.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"data\"")
			}
		case "error":
			if err := func() error {
				s.Error.Reset()
				if err := s.Error.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"error\"")
			}
		case "errorText":
			if err := func() error {
				s.ErrorText.Reset()
				if err := s.ErrorText.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"errorText\"")
			}
		case "additionalErrors":
			if err := func() error {
				s.AdditionalErrors.Reset()
				if err := s.AdditionalErrors.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"additionalErrors\"")
			}
		default:
			return d.Skip()
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "decode WbFeedbackResponse")
	}

	return nil
}

func (s *WbFeedbackResponse) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
