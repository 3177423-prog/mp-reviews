package consts

import (
	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

// Направление сортировки.
type ReviewListSortDirType string

type OptReviewListSortDirType struct {
	Value ReviewListSortDirType
	Set   bool
}

const (
	// `ASC` — по возрастанию.
	ReviewListSortDirASC ReviewListSortDirType = "ASC"
	// `DESC` — по убыванию.
	ReviewListSortDirDESC ReviewListSortDirType = "DESC"
)

func (ReviewListSortDirType) AllValues() []ReviewListSortDirType {
	return []ReviewListSortDirType{
		ReviewListSortDirASC,
		ReviewListSortDirDESC,
	}
}

func (s ReviewListSortDirType) MarshalText() ([]byte, error) {
	switch s {
	case ReviewListSortDirASC:
		return []byte(s), nil
	case ReviewListSortDirDESC:
		return []byte(s), nil
	default:
		return nil, errors.Errorf("invalid value: %q", s)
	}
}

func (s *ReviewListSortDirType) UnmarshalText(data []byte) error {
	switch ReviewListSortDirType(data) {
	case ReviewListSortDirASC:
		*s = ReviewListSortDirASC
		return nil
	case ReviewListSortDirDESC:
		*s = ReviewListSortDirDESC
		return nil
	default:
		return errors.Errorf("invalid value: %q", data)
	}
}

func (s ReviewListSortDirType) Validate() error {
	switch s {
	case "ASC":
		return nil
	case "DESC":
		return nil
	default:
		return errors.Errorf("invalid value: %v", s)
	}
}

func (s ReviewListSortDirType) Encode(e *jx.Encoder) {
	e.Str(string(s))
}

func (s *ReviewListSortDirType) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode ReviewListSortDirType to nil")
	}
	v, err := d.StrBytes()
	if err != nil {
		return err
	}
	// Try to use constant string.
	switch ReviewListSortDirType(v) {
	case ReviewListSortDirASC:
		*s = ReviewListSortDirASC
	case ReviewListSortDirDESC:
		*s = ReviewListSortDirDESC
	default:
		*s = ReviewListSortDirType(v)
	}

	return nil
}

func (s ReviewListSortDirType) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *ReviewListSortDirType) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}

func (o OptReviewListSortDirType) IsSet() bool { return o.Set }

func (o *OptReviewListSortDirType) Reset() {
	var v ReviewListSortDirType
	o.Value = v
	o.Set = false
}

func (o OptReviewListSortDirType) Get() (v ReviewListSortDirType, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptReviewListSortDirType) Or(d ReviewListSortDirType) ReviewListSortDirType {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptReviewListSortDirType) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	e.Str(string(o.Value))
}

func (o *OptReviewListSortDirType) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("invalid: unable to decode OptReviewListSortDirType to nil")
	}
	o.Set = true
	if err := o.Value.Decode(d); err != nil {
		return err
	}
	return nil
}

func (s OptReviewListSortDirType) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *OptReviewListSortDirType) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
