package domain

import "github.com/go-faster/jx"

type ReviewCommentCreateRequest struct {
	// Обновление статуса у отзыва.
	MarkReviewAsProcessed OptBool `json:"mark_review_as_processed"`
	// Идентификатор родительского комментария.
	ParentCommentID OptString `json:"parent_comment_id"`
	// Идентификатор отзыва.
	ReviewID string `json:"review_id"`
	// Текст комментария.
	Text string `json:"text"`
}

func (s *ReviewCommentCreateRequest) Encode(e *jx.Encoder) {
	e.ObjStart()
	s.encodeFields(e)
	e.ObjEnd()
}

// encodeFields encodes fields.
func (s *ReviewCommentCreateRequest) encodeFields(e *jx.Encoder) {
	if s.MarkReviewAsProcessed.Set {
		e.FieldStart("mark_review_as_processed")
		s.MarkReviewAsProcessed.Encode(e)
	}
	if s.ParentCommentID.Set {
		e.FieldStart("parent_comment_id")
		s.ParentCommentID.Encode(e)
	}
	{
		e.FieldStart("review_id")
		e.Str(s.ReviewID)
	}

	{
		e.FieldStart("text")
		e.Str(s.Text)
	}
}

func (s *ReviewCommentCreateRequest) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}
