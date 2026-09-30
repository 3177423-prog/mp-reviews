package domain

import (
	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

// Информация по отзывам
type WbFeedbacksData struct {
	CountUnanswered uint32             `json:"countUnanswered"` // Количество необработанных отзывов
	CountArchive    uint32             `json:"countArchive"`    // Количество обработанных отзывов
	Feedbacks       OptWbFeedbackArray `json:"feedbacks"`       // Массив отзывов на товары
}

func (s *WbFeedbacksData) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode WbFeedbacksData to nil")
	}

	if err := d.ObjBytes(func(d *jx.Decoder, k []byte) error {
		switch string(k) {
		case "countUnanswered":
			if err := func() error {
				v, err := d.UInt32()
				s.CountUnanswered = uint32(v)
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"countUnanswered\"")
			}
		case "countArchive":
			if err := func() error {
				v, err := d.UInt32()
				s.CountArchive = uint32(v)
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"countArchive\"")
			}
		case "feedbacks":
			if err := func() error {
				if err := s.Feedbacks.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"feedbacks\"")
			}
		default:
			return d.Skip()
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "decode WbFeedbacksData")
	}

	return nil
}

func (s *WbFeedbacksData) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
