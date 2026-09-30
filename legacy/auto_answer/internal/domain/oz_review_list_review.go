package domain

import (
	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
	"github.com/ogen-go/ogen/json"

	"auto_answer/internal/consts"
)

// Информация об отзыве.
type ReviewListReview struct {
	// Количество комментариев у отзыва.
	CommentsAmount OptInt32 `json:"comments_amount"`
	// Идентификатор отзыва.
	ID OptString `json:"id"`
	// true, если отзыв участвует в подсчёте рейтинга.
	IsRatingParticipant OptBool `json:"is_rating_participant"`
	// Статус заказа, на который покупатель оставил отзыв.
	OrderStatus consts.OptReviewListOrderStatusType `json:"order_status"`
	// Количество изображений у отзыва.
	PhotosAmount OptInt32 `json:"photos_amount"`
	// Дата публикации отзыва.
	PublishedAt OptDateTime `json:"published_at"`
	// Оценка отзыва.
	Rating OptUInt8 `json:"rating"`
	// Идентификатор товара в системе Ozon.
	Sku OptInt64 `json:"sku"`
	// Статус отзыва.
	Status consts.OptReviewListStatusType `json:"status"`
	// Текст отзыва.
	Text OptString `json:"text"`
	// Количество видео у отзыва.
	VideosAmount OptInt32 `json:"videos_amount"`
}

type ReviewListReviewArray []ReviewListReview

type OptReviewListReviewArray struct {
	Value ReviewListReviewArray
	Set   bool
}

func (s *ReviewListReview) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode ReviewListReview to nil")
	}

	if err := d.ObjBytes(func(d *jx.Decoder, k []byte) error {
		switch string(k) {
		case "comments_amount":
			if err := func() error {
				s.CommentsAmount.Reset()
				if err := s.CommentsAmount.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"comments_amount\"")
			}
		case "id":
			if err := func() error {
				s.ID.Reset()
				if err := s.ID.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"id\"")
			}
		case "is_rating_participant":
			if err := func() error {
				s.IsRatingParticipant.Reset()
				if err := s.IsRatingParticipant.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"is_rating_participant\"")
			}
		case "order_status":
			if err := func() error {
				s.OrderStatus.Reset()
				if err := s.OrderStatus.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"order_status\"")
			}
		case "photos_amount":
			if err := func() error {
				s.PhotosAmount.Reset()
				if err := s.PhotosAmount.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"photos_amount\"")
			}
		case "published_at":
			if err := func() error {
				s.PublishedAt.Reset()
				if err := s.PublishedAt.Decode(d, json.DecodeDateTime); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"published_at\"")
			}
		case "rating":
			if err := func() error {
				s.Rating.Reset()
				if err := s.Rating.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"rating\"")
			}
		case "sku":
			if err := func() error {
				s.Sku.Reset()
				if err := s.Sku.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"sku\"")
			}
		case "status":
			if err := func() error {
				s.Status.Reset()
				if err := s.Status.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"status\"")
			}
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
		case "videos_amount":
			if err := func() error {
				s.VideosAmount.Reset()
				if err := s.VideosAmount.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"videos_amount\"")
			}
		default:
			return d.Skip()
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "decode ReviewListReview")
	}

	return nil
}

func (s *ReviewListReview) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}

func (o OptReviewListReviewArray) IsSet() bool { return o.Set }

func (o *OptReviewListReviewArray) Reset() {
	var v ReviewListReviewArray
	o.Value = v
	o.Set = false
}

func (o OptReviewListReviewArray) Or(d ReviewListReviewArray) ReviewListReviewArray {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptReviewListReviewArray) Get() (v ReviewListReviewArray, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o *OptReviewListReviewArray) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("invalid: unable to decode OptReviewListReviewArray to nil")
	}
	if d.Next() == jx.Null {
		if err := d.Null(); err != nil {
			return err
		}

		var v ReviewListReviewArray
		o.Value = v
		o.Set = true
		return nil
	}
	o.Set = true
	o.Value = make(ReviewListReviewArray, 0)
	if err := d.Arr(func(d *jx.Decoder) error {
		var elem ReviewListReview
		err := elem.Decode(d)
		if err != nil {
			return err
		}
		o.Value = append(o.Value, elem)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *OptReviewListReviewArray) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
