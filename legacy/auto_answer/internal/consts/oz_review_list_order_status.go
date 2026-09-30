package consts

import (
	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

// Статус заказа, на который покупатель оставил отзыв.
type ReviewListOrderStatusType string

type OptReviewListOrderStatusType struct {
	Value ReviewListOrderStatusType
	Set   bool
}

const (
	// `DELIVERED` — доставлен.
	ReviewListOrderStatusDelivered ReviewListOrderStatusType = "DELIVERED"
	// `CANCELLED` — отменён.
	ReviewListOrderStatusCancelled ReviewListOrderStatusType = "CANCELLED"
)

func (ReviewListOrderStatusType) AllValues() []ReviewListOrderStatusType {
	return []ReviewListOrderStatusType{
		ReviewListOrderStatusDelivered,
		ReviewListOrderStatusCancelled,
	}
}

func (s ReviewListOrderStatusType) MarshalText() ([]byte, error) {
	switch s {
	case ReviewListOrderStatusDelivered:
		return []byte(s), nil
	case ReviewListOrderStatusCancelled:
		return []byte(s), nil
	default:
		return nil, errors.Errorf("invalid value: %q", s)
	}
}

func (s *ReviewListOrderStatusType) UnmarshalText(data []byte) error {
	switch ReviewListOrderStatusType(data) {
	case ReviewListOrderStatusDelivered:
		*s = ReviewListOrderStatusDelivered
		return nil
	case ReviewListOrderStatusCancelled:
		*s = ReviewListOrderStatusCancelled
		return nil
	default:
		return errors.Errorf("invalid value: %q", data)
	}
}

func (s ReviewListOrderStatusType) Validate() error {
	switch s {
	case "DELIVERED":
		return nil
	case "CANCELLED":
		return nil
	default:
		return errors.Errorf("invalid value: %v", s)
	}
}

func (s ReviewListOrderStatusType) Encode(e *jx.Encoder) {
	e.Str(string(s))
}

func (s *ReviewListOrderStatusType) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode ReviewListOrderStatusType to nil")
	}
	v, err := d.StrBytes()
	if err != nil {
		return err
	}
	// Try to use constant string.
	switch ReviewListOrderStatusType(v) {
	case ReviewListOrderStatusDelivered:
		*s = ReviewListOrderStatusDelivered
	case ReviewListOrderStatusCancelled:
		*s = ReviewListOrderStatusCancelled
	default:
		*s = ReviewListOrderStatusType(v)
	}

	return nil
}

func (s ReviewListOrderStatusType) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *ReviewListOrderStatusType) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}

func (o OptReviewListOrderStatusType) IsSet() bool { return o.Set }

func (o *OptReviewListOrderStatusType) Reset() {
	var v ReviewListOrderStatusType
	o.Value = v
	o.Set = false
}

func (o OptReviewListOrderStatusType) Get() (v ReviewListOrderStatusType, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptReviewListOrderStatusType) Or(d ReviewListOrderStatusType) ReviewListOrderStatusType {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptReviewListOrderStatusType) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	e.Str(string(o.Value))
}

func (o *OptReviewListOrderStatusType) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("invalid: unable to decode OptReviewListOrderStatusType to nil")
	}
	o.Set = true
	if err := o.Value.Decode(d); err != nil {
		return err
	}
	return nil
}

func (s OptReviewListOrderStatusType) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *OptReviewListOrderStatusType) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
