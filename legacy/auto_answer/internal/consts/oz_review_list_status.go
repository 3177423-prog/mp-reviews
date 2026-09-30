package consts

import (
	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

// Статус отзыва
type ReviewListStatusType string

type OptReviewListStatusType struct {
	Value ReviewListStatusType
	Set   bool
}

const (
	// `ALL` — все.
	ReviewListStatusAll ReviewListStatusType = "ALL"
	// `UNPROCESSED` — необработанные.
	ReviewListStatusUnprocessed ReviewListStatusType = "UNPROCESSED"
	// `PROCESSED` — обработанные.
	ReviewListStatusProcessed ReviewListStatusType = "PROCESSED"
)

func (ReviewListStatusType) AllValues() []ReviewListStatusType {
	return []ReviewListStatusType{
		ReviewListStatusAll,
		ReviewListStatusUnprocessed,
		ReviewListStatusProcessed,
	}
}

func (s ReviewListStatusType) MarshalText() ([]byte, error) {
	switch s {
	case ReviewListStatusAll:
		return []byte(s), nil
	case ReviewListStatusUnprocessed:
		return []byte(s), nil
	case ReviewListStatusProcessed:
		return []byte(s), nil
	default:
		return nil, errors.Errorf("invalid value: %q", s)
	}
}

func (s *ReviewListStatusType) UnmarshalText(data []byte) error {
	switch ReviewListStatusType(data) {
	case ReviewListStatusAll:
		*s = ReviewListStatusAll
		return nil
	case ReviewListStatusUnprocessed:
		*s = ReviewListStatusUnprocessed
		return nil
	case ReviewListStatusProcessed:
		*s = ReviewListStatusProcessed
		return nil
	default:
		return errors.Errorf("invalid value: %q", data)
	}
}

func (s ReviewListStatusType) Validate() error {
	switch s {
	case "ALL":
		return nil
	case "UNPROCESSED":
		return nil
	case "PROCESSED":
		return nil
	default:
		return errors.Errorf("invalid value: %v", s)
	}
}

// Encode encodes AgeUnitType as json.
func (s ReviewListStatusType) Encode(e *jx.Encoder) {
	e.Str(string(s))
}

// Decode decodes AgeUnitType from json.
func (s *ReviewListStatusType) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode ReviewListStatusType to nil")
	}
	v, err := d.StrBytes()
	if err != nil {
		return err
	}
	// Try to use constant string.
	switch ReviewListStatusType(v) {
	case ReviewListStatusAll:
		*s = ReviewListStatusAll
	case ReviewListStatusUnprocessed:
		*s = ReviewListStatusUnprocessed
	case ReviewListStatusProcessed:
		*s = ReviewListStatusProcessed
	default:
		*s = ReviewListStatusType(v)
	}

	return nil
}

// MarshalJSON implements stdjson.Marshaler.
func (s ReviewListStatusType) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

// UnmarshalJSON implements stdjson.Unmarshaler.
func (s *ReviewListStatusType) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}

func (o OptReviewListStatusType) IsSet() bool { return o.Set }

func (o *OptReviewListStatusType) Reset() {
	var v ReviewListStatusType
	o.Value = v
	o.Set = false
}

func (o OptReviewListStatusType) Get() (v ReviewListStatusType, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptReviewListStatusType) Or(d ReviewListStatusType) ReviewListStatusType {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptReviewListStatusType) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	e.Str(string(o.Value))
}

func (o *OptReviewListStatusType) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("invalid: unable to decode OptReviewListStatusType to nil")
	}
	o.Set = true
	if err := o.Value.Decode(d); err != nil {
		return err
	}
	return nil
}

func (s OptReviewListStatusType) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *OptReviewListStatusType) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
