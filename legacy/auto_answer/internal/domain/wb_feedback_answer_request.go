package domain

import "github.com/go-faster/jx"

type WbFeedbackAnswerRequest struct {
	// ID отзыва.
	ID string `json:"id"`
	// Текст ответа. Минимум 2, максимум 5000 символов
	Text string `json:"text"`
}

func (s *WbFeedbackAnswerRequest) Encode(e *jx.Encoder) {
	e.ObjStart()
	s.encodeFields(e)
	e.ObjEnd()
}

// encodeFields encodes fields.
func (s *WbFeedbackAnswerRequest) encodeFields(e *jx.Encoder) {
	{
		e.FieldStart("id")
		e.Str(s.ID)
	}
	{
		e.FieldStart("text")
		e.Str(s.Text)
	}
}

func (s *WbFeedbackAnswerRequest) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}
