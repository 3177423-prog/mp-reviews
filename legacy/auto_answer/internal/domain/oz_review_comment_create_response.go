package domain

import (
	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

type ReviewCommentCreateResponse struct {
	// Идентификатор комментария.
	CommentID OptString `json:"comment_id"`
}

func (s *ReviewCommentCreateResponse) Decode(d *jx.Decoder) error {
	if s == nil {
		return errors.New("invalid: unable to decode ReviewCommentCreateResponse to nil")
	}

	if err := d.ObjBytes(func(d *jx.Decoder, k []byte) error {
		switch string(k) {
		case "comment_id":
			if err := func() error {
				s.CommentID.Reset()
				if err := s.CommentID.Decode(d); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return errors.Wrap(err, "decode field \"comment_id\"")
			}
		default:
			return d.Skip()
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "decode ReviewCommentCreateResponse")
	}

	return nil
}

func (s *ReviewCommentCreateResponse) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
