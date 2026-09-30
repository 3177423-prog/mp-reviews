package domain

import (
	"github.com/go-faster/jx"

	"auto_answer/internal/consts"
)

type ReviewListRequest struct {
	// Идентификатор последнего отзыва на странице.
	LastID OptString `json:"last_id"`
	// Количество отзывов в ответе. Минимум — 20, максимум — 100.
	Limit int32 `json:"limit"`
	// Направление сортировки
	SortDir consts.OptReviewListSortDirType `json:"sort_dir"`
	// Статусы отзывов
	Status consts.OptReviewListStatusType `json:"status"`
}

func (s *ReviewListRequest) Encode(e *jx.Encoder) {
	e.ObjStart()
	s.encodeFields(e)
	e.ObjEnd()
}

func (s *ReviewListRequest) encodeFields(e *jx.Encoder) {
	if s.LastID.Set {
		e.FieldStart("last_id")
		s.LastID.Encode(e)
	}
	{
		e.FieldStart("limit")
		e.Int32(s.Limit)
	}
	if s.SortDir.Set {
		e.FieldStart("sort_dir")
		s.SortDir.Encode(e)
	}
	if s.Status.Set {
		e.FieldStart("status")
		s.Status.Encode(e)
	}

}

func (s *ReviewListRequest) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}
