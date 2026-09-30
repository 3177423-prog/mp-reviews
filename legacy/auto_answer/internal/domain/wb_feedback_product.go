package domain

import (
	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

// Детали по товару, на который оставлен отзыв
type WbFeedbackProduct struct {
	NmID        int64        `json:"nmId"`            // Числовой идентификатор номенклатуры Wildberries
	ImtID       int64        `json:"imtId"`           // Идентификатор карточки товара
	Title       string       `json:"productName"`     // Название товара
	Article     OptNilString `json:"supplierArticle"` // Артикул продавца
	SellerTitle OptNilString `json:"supplierName"`    // Имя продавца
	Brand       OptNilString `json:"brandName"`       // Бренд товара
	TechSize    OptNilString `json:"size"`            // Размер товара (techSize в КТ)
}

func (s *WbFeedbackProduct) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode WbFeedbackProduct to nil")
	}

	if err := d.ObjBytes(func(d *jx.Decoder, k []byte) error {
		switch string(k) {
		case "nmId":
			if err := func() error {
				v, err := d.Int64()
				s.NmID = int64(v)
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"nmId\"")
			}
		case "imtId":
			if err := func() error {
				v, err := d.Int64()
				s.ImtID = int64(v)
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"imtId\"")
			}
		case "productName":
			if err := func() error {
				v, err := d.Str()
				s.Title = string(v)
				if err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"productName\"")
			}
		case "supplierArticle":
			if err := func() error {
				s.Article.Reset()
				if err := s.Article.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"supplierArticle\"")
			}
		case "supplierName":
			if err := func() error {
				s.SellerTitle.Reset()
				if err := s.SellerTitle.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"supplierName\"")
			}
		case "brandName":
			if err := func() error {
				s.Brand.Reset()
				if err := s.Brand.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"brandName\"")
			}
		case "size":
			if err := func() error {
				s.TechSize.Reset()
				if err := s.TechSize.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"size\"")
			}
		default:
			return d.Skip()
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "decode WbFeedbackProduct")
	}

	return nil
}

func (s *WbFeedbackProduct) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
