package domain

import (
	"time"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
	"github.com/ogen-go/ogen/json"

	"auto_answer/internal/consts"
)

// Отзыв на товар
type WbFeedbackItem struct {
	ID               string                                  `json:"id"`               // ID отзыва
	Text             OptString                               `json:"text"`             // Текст отзыва
	Advantages       OptString                               `json:"pros"`             // Достоинства товара
	Defects          OptString                               `json:"cons"`             // Недостатки товара
	ProductValuation uint8                                   `json:"productValuation"` // Оценка товара
	CreatedDate      time.Time                               `json:"createdDate"`      // Дата и время создания отзыва
	Answer           OptNilWbFeedbackAnswer                  `json:"answer"`           // Ответ на отзыв
	State            consts.WbFeedbackStateType              `json:"state"`            // Статус отзыва
	ProductDetails   WbFeedbackProduct                       `json:"productDetails"`   // Информация о товаре
	PhotoLinks       OptNilWbFeedbackPhotoArray              `json:"photoLinks"`       // Приложенные к отзыву фотографии
	WasViewed        bool                                    `json:"wasViewed"`        // Просмотрен ли отзыв
	UserName         OptNilString                            `json:"userName"`         // Имя автора отзыва
	MatchingSize     consts.OptNilWbFeedbackMatchingSizeType `json:"matchingSize"`     // Соответствие заявленного размера реальному
	ParentFeedbackID OptNilString                            `json:"parentFeedbackId"` // ID начального отзыва (null, если этот отзыв начальный)
}

// Массив структур отзывов
type WbFeedbackArray []WbFeedbackItem

type OptWbFeedbackArray struct {
	Value WbFeedbackArray
	Set   bool
}

func (s *WbFeedbackItem) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode WbFeedbackItem to nil")
	}

	if err := d.ObjBytes(func(d *jx.Decoder, k []byte) error {
		switch string(k) {
		case "id":
			if err := func() error {
				v, err := d.Str()
				s.ID = string(v)
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"id\"")
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
		case "pros":
			if err := func() error {
				s.Advantages.Reset()
				if err := s.Advantages.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"pros\"")
			}
		case "cons":
			if err := func() error {
				s.Defects.Reset()
				if err := s.Defects.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"cons\"")
			}
		case "productValuation":
			if err := func() error {
				v, err := d.UInt8()
				s.ProductValuation = uint8(v)
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"productValuation\"")
			}
		case "createdDate":
			if err := func() error {
				v, err := json.DecodeDateTime(d)
				s.CreatedDate = v
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"createdDate\"")
			}
		case "answer":
			if err := func() error {
				s.Answer.Reset()
				if err := s.Answer.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"answer\"")
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
		case "productDetails":
			if err := func() error {
				if err := s.ProductDetails.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"productDetails\"")
			}
		case "photoLinks":
			if err := func() error {
				s.PhotoLinks.Reset()
				if err := s.PhotoLinks.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"photoLinks\"")
			}
		case "wasViewed":
			if err := func() error {
				v, err := d.Bool()
				s.WasViewed = bool(v)
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"wasViewed\"")
			}
		case "userName":
			if err := func() error {
				s.UserName.Reset()
				if err := s.UserName.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"userName\"")
			}
		case "matchingSize":
			if err := func() error {
				s.MatchingSize.Reset()
				if err := s.MatchingSize.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"matchingSize\"")
			}
		case "parentFeedbackId":
			if err := func() error {
				s.ParentFeedbackID.Reset()
				if err := s.ParentFeedbackID.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"parentFeedbackId\"")
			}
		default:
			return d.Skip()
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "decode WbFeedbackItem")
	}

	return nil
}

func (s *WbFeedbackItem) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}

func (o OptWbFeedbackArray) IsSet() bool { return o.Set }

func (o *OptWbFeedbackArray) Reset() {
	var v WbFeedbackArray
	o.Value = v
	o.Set = false
}

func (o OptWbFeedbackArray) Or(d WbFeedbackArray) WbFeedbackArray {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptWbFeedbackArray) Get() (v WbFeedbackArray, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o *OptWbFeedbackArray) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("invalid: unable to decode OptWbFeedbackArray to nil")
	}
	if d.Next() == jx.Null {
		if err := d.Null(); err != nil {
			return err
		}

		var v WbFeedbackArray
		o.Value = v
		o.Set = true
		return nil
	}
	o.Set = true
	o.Value = make(WbFeedbackArray, 0)
	if err := d.Arr(func(d *jx.Decoder) error {
		var elem WbFeedbackItem
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

func (s *OptWbFeedbackArray) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
