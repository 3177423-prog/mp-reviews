package parameters

import (
	"auto_answer/internal/consts"
	"auto_answer/internal/domain"
)

// Параметры для вызова метода
// GET https://feedbacks-api.wildberries.ru/api/v1/feedbacks
type WbFeedbackList struct {
	// Обработанные отзывы (true) или необработанные отзывы(false)
	IsAnswered bool
	// Артикул WB
	NmID domain.OptInt64
	// Количество отзывов (max. 5 000)
	Take uint32
	// Количество отзывов для пропуска (max. 199990)
	Skip uint32
	// Сортировка отзывов по дате
	Order consts.OptWbOrderFeedbackListType
	// Дата начала периода в формате Unix timestamp
	DateFrom domain.OptDateTime
	// Дата конца периода в формате Unix timestamp
	DateTo domain.OptDateTime
}
