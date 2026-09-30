package models

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WbFeedback struct {
	SellerID     int16       // ID продавца
	WbID         string      // ID отзыва WB
	FeedbackText pgtype.Text // Текст отзыва
	Advantages   pgtype.Text // Достоинства товара
	Defects      pgtype.Text // Недостатки товара
	Valuation    uint8       // Оценка товара: 1-5
	CreatedAt    time.Time   // Дата оценки
	NmID         int64       // Артикул WB
	ImtID        int64       // ID карточки товара
	TechSize     pgtype.Text // Размер товара
	UserName     pgtype.Text // Имя автора отзыва
	MatchingSize pgtype.Text // Соответствие заявленного размера реальному. Возможные значения: null - для безразмерных товаров, 'ок' - соответствует размеру, 'smaller' - маломерит, 'bigger' - большемерит
	ParentWbID   pgtype.Text // ID начального отзыва
	Article      pgtype.Text // Артикул товара
}
type WbFeedbacks []WbFeedback

// Получаем данные
func (fb *WbFeedback) Get(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	query := `SELECT seller_id, wb_id, feedback_text, advantages, defects, valuation, created_at, nm_id, imt_id, tech_size,
	                 user_name, matching_size, parent_id, article
            FROM wb.feedbacks
						WHERE wb_id = @wbID`
	args := pgx.NamedArgs{"wbID": fb.WbID}
	row := conn.QueryRow(context.Background(), query, args)

	err = row.Scan(&fb.SellerID, &fb.WbID, &fb.FeedbackText, &fb.Advantages, &fb.Defects, &fb.Valuation, &fb.CreatedAt, &fb.NmID, &fb.ImtID, &fb.TechSize,
		&fb.UserName, &fb.MatchingSize, &fb.ParentWbID, &fb.Article)
	if err != nil {
		return err
	}

	return nil
}

// Вставляем новую запись в БД
func (fb *WbFeedback) Insert(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	query := `INSERT INTO wb.feedbacks
	            (seller_id, wb_id, feedback_text, advantages, defects, valuation, created_at, nm_id, imt_id, tech_size, user_name, matching_size, parent_id, article)
	          VALUES
						  (@sellerID, @wbID, @feedbackText, @advantages, @defects, @valuation, @createdAt, @nmID, @imtID, @techSize, @userName, @matchingSize, @parentID, @article)`
	args := pgx.NamedArgs{
		"sellerID":     fb.SellerID,
		"wbID":         fb.WbID,
		"feedbackText": fb.FeedbackText,
		"advantages":   fb.Advantages,
		"defects":      fb.Defects,
		"valuation":    fb.Valuation,
		"createdAt":    fb.CreatedAt,
		"nmID":         fb.NmID,
		"imtID":        fb.ImtID,
		"techSize":     fb.TechSize,
		"userName":     fb.UserName,
		"matchingSize": fb.MatchingSize,
		"parentID":     fb.ParentWbID,
		"article":      fb.Article,
	}
	ct, err := conn.Exec(context.Background(), query, args)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return errors.New("ошибка при добавлении записи")
	}

	return nil
}

// Добавляем список
func (fbs *WbFeedbacks) BulkInsert(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	query := `INSERT INTO wb.feedbacks
	            (seller_id, wb_id, feedback_text, advantages, defects, valuation, created_at, nm_id, imt_id, tech_size, user_name, matching_size, parent_id, article)
	          VALUES
						  (@sellerID, @wbID, @feedbackText, @advantages, @defects, @valuation, @createdAt, @nmID, @imtID, @techSize, @userName, @matchingSize, @parentID, @article)`

	batch := &pgx.Batch{}
	for _, fb := range *fbs {
		args := pgx.NamedArgs{
			"sellerID":     fb.SellerID,
			"wbID":         fb.WbID,
			"feedbackText": fb.FeedbackText,
			"advantages":   fb.Advantages,
			"defects":      fb.Defects,
			"valuation":    fb.Valuation,
			"createdAt":    fb.CreatedAt,
			"nmID":         fb.NmID,
			"imtID":        fb.ImtID,
			"techSize":     fb.TechSize,
			"userName":     fb.UserName,
			"matchingSize": fb.MatchingSize,
			"parentID":     fb.ParentWbID,
			"article":      fb.Article,
		}
		batch.Queue(query, args)
	}

	results := conn.SendBatch(context.Background(), batch)
	defer results.Close()

	for _, fb := range *fbs {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("невозможно вставить строку - nmID: %d, wbID: '%s', date: '%s': %w", fb.NmID, fb.WbID, fb.CreatedAt.Format("2006-01-02"), err)
		}
	}

	return results.Close()
}

// Обновляем запись в БД
func (fb *WbFeedback) Update(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	query := `UPDATE wb.feedbacks
            SET feedback_text = @feedbackText, advantages = @advantages, defects = @defects, valuation = @valuation, imt_id = @imtID,
						    tech_size = @techSize, user_name = @userName, matching_size = @matchingSize, parent_id = @parentID, article = @article
						WHERE wb_id = @wbID`
	args := pgx.NamedArgs{
		"wbID":         fb.WbID,
		"feedbackText": fb.FeedbackText,
		"advantages":   fb.Advantages,
		"defects":      fb.Defects,
		"valuation":    fb.Valuation,
		"imtID":        fb.ImtID,
		"techSize":     fb.TechSize,
		"userName":     fb.UserName,
		"matchingSize": fb.MatchingSize,
		"parentID":     fb.ParentWbID,
		"article":      fb.Article,
	}
	ct, err := conn.Exec(context.Background(), query, args)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return errors.New("нет записей с таким NmID и ChrtID, нечего обновлять")
	}

	return nil
}

// Удяляем запись из БД
func (fb *WbFeedback) Delete(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	query := `DELETE FROM wb.feedbacks
            WHERE wb_id = @wbID`
	args := pgx.NamedArgs{"wbID": fb.WbID}
	ct, err := conn.Exec(context.Background(), query, args)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return errors.New("нет записей с таким ID, нечего удалять")
	}

	return nil
}
