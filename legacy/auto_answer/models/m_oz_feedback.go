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

// Информация об отзыве.
type OzonFeedback struct {
	SellerID         int16       // ID продавца
	OzonID           string      // Идентификатор отзыва
	FeedbackText     pgtype.Text // Текст отзыва
	Valuation        uint8       // Оценка товара: 1-5
	CreatedAt        time.Time   // Дата оценки
	ProductID        int64       // Внутренний идентификатор товара Озон
	Sku              int64       // Идентификатор товара в системе Ozon
	IsRating         bool        // Признак участия в подсчёте рейтинга
	IsOrderDelivered bool        // Признак, что заказ был доставлен (не отменен)
}
type OzonFeedbacks []OzonFeedback

// Получаем данные
func (f *OzonFeedback) Get(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	query := `SELECT seller_id, oz_id, feedback_text, valuation, created_at, product_id, sku, is_rating, is_delivered
            FROM oz.feedbacks
						WHERE oz_id = @ozID`
	args := pgx.NamedArgs{"ozID": f.OzonID}
	row := conn.QueryRow(context.Background(), query, args)

	err = row.Scan(&f.SellerID, &f.OzonID, &f.FeedbackText, &f.Valuation, &f.CreatedAt, &f.ProductID, &f.Sku, &f.IsRating, &f.IsOrderDelivered)
	if err != nil {
		return err
	}

	return nil
}

// Вставляем новую запись в БД
func (f *OzonFeedback) Insert(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	query := `INSERT INTO oz.feedbacks
	            (seller_id, oz_id, feedback_text, valuation, created_at, product_id, sku, is_rating, is_delivered)
	          VALUES
						  (@sellerID, @ozID, @feedbackText, @valuation, @createdAt, @productID, @sku, @isRating, @isDelivered)`
	args := pgx.NamedArgs{
		"sellerID":     f.SellerID,
		"ozID":         f.OzonID,
		"feedbackText": f.FeedbackText,
		"valuation":    f.Valuation,
		"createdAt":    f.CreatedAt,
		"productID":    f.ProductID,
		"sku":          f.Sku,
		"isRating":     f.IsRating,
		"isDelivered":  f.IsOrderDelivered,
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
func (fs *OzonFeedbacks) BulkInsert(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	query := `INSERT INTO oz.feedbacks
	            (seller_id, oz_id, feedback_text, valuation, created_at, product_id, sku, is_rating, is_delivered)
	          VALUES
						  (@sellerID, @ozID, @feedbackText, @valuation, @createdAt, @productID, @sku, @isRating, @isDelivered)`

	batch := &pgx.Batch{}
	for _, f := range *fs {
		args := pgx.NamedArgs{
			"sellerID":     f.SellerID,
			"ozID":         f.OzonID,
			"feedbackText": f.FeedbackText,
			"valuation":    f.Valuation,
			"createdAt":    f.CreatedAt,
			"productID":    f.ProductID,
			"sku":          f.Sku,
			"isRating":     f.IsRating,
			"isDelivered":  f.IsOrderDelivered,
		}
		batch.Queue(query, args)
	}

	results := conn.SendBatch(context.Background(), batch)
	defer results.Close()

	for _, f := range *fs {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("невозможно вставить строку - sku: %d, ozonID: '%s', date: '%s': %w", f.Sku, f.OzonID, f.CreatedAt.Format("2006-01-02"), err)
		}
	}

	return results.Close()
}

// Обновляем запись в БД
func (f *OzonFeedback) Update(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	query := `UPDATE oz.feedbacks
            SET feedback_text = @feedbackText, valuation = @valuation, is_rating = @isRating
						WHERE oz_id = @ozID`
	args := pgx.NamedArgs{
		"ozID":         f.OzonID,
		"feedbackText": f.FeedbackText,
		"valuation":    f.Valuation,
		"isRating":     f.IsRating,
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
func (f *OzonFeedback) Delete(db *pgxpool.Pool) error {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	query := `DELETE FROM oz.feedbacks
            WHERE oz_id = @ozID`
	args := pgx.NamedArgs{"ozID": f.OzonID}
	ct, err := conn.Exec(context.Background(), query, args)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return errors.New("нет записей с таким ID, нечего удалять")
	}

	return nil
}
