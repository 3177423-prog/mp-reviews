package domain

import (
	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

type ReviewListResponse struct {
	// true, если в ответе вернули не все отзывы.
	HasNext OptBool `json:"has_next"`
	// Идентификатор последнего отзыва на странице.
	LastID OptString `json:"last_id"`
	// Информация об отзыве.
	Reviews OptReviewListReviewArray `json:"reviews"`
}

func (s *ReviewListResponse) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode ReviewListResponse to nil")
	}

	if err := d.ObjBytes(func(d *jx.Decoder, k []byte) error {
		switch string(k) {
		case "has_next":
			if err := func() error {
				s.HasNext.Reset()
				if err := s.HasNext.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"has_next\"")
			}
		case "last_id":
			if err := func() error {
				s.LastID.Reset()
				if err := s.LastID.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"last_id\"")
			}
		case "reviews":
			if err := func() error {
				s.Reviews.Reset()
				if err := s.Reviews.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"reviews\"")
			}

		default:
			return d.Skip()
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "decode ReviewListResponse")
	}

	return nil
}

func (s *ReviewListResponse) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
