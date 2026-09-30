package models

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Рейтинг, сохраненный в БД
type DbRate struct {
	ImtID          int64     // Идентификатор номенклатуры товара (nmID) либо карточки товара (imtID)
	DateRate       time.Time // Дата сохранения оценки
	SumValuation   int32     // Сумма всех оценок
	CountValuation int32     // Количество оценок
	Rate           float64   // Средняя оценка
	SellerID       int16     // Идентификатор поставщика
}

// Рейтинг по номенклатуре
type ProductRate struct {
	NmID           int64   // Идентификатор номенклатуры товара
	ImtID          int64   // Идентификатор карточки товара
	Name           string  // Наименование товара
	Rate           float64 // Средняя оценка карточки товара по всем номенклатурам
	CountValuation int32   // Количество оценок
}

type ProductRates []ProductRate
type MapProductRate map[int64]ProductRate

func SelectAllImts(sellerID int16) (MapProductRate, error) {
	result := MapProductRate{}

	conn, err := DBNew.Acquire(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	query := `SELECT p.nm_id, p.imt_id, p.title
	          FROM wb.products p
						WHERE p.seller_id = @sellerID AND p.deleted_at IS NULL`
	args := pgx.NamedArgs{"sellerID": sellerID}
	rows, _ := conn.Query(context.Background(), query, args)
	for rows.Next() {
		var pr ProductRate
		err = rows.Scan(&pr.NmID, &pr.ImtID, &pr.Name)
		if err != nil {
			return nil, err
		}

		if _, seek := result[pr.ImtID]; !seek {
			result[pr.ImtID] = pr
		}
	}

	return result, rows.Err()
}

// Получаем сохраненные ранее в БД данные, у которых оценка выше или равна заданной
func SelectAllBDRatings(threshold float64, sellerID int16, dt time.Time) (map[int64]DbRate, error) {
	rates := map[int64]DbRate{}
	conn, err := DBNew.Acquire(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	query := `SELECT imt_id, sum_valuation, count_valuation, rate
		        FROM wb.product_rates
		        WHERE seller_id = @sellerID AND date_rate = @dateRate AND rate >= @rate`
	args := pgx.NamedArgs{"sellerID": sellerID, "dateRate": dt, "rate": threshold}
	rows, _ := conn.Query(context.Background(), query, args)
	for rows.Next() {
		var r DbRate
		err = rows.Scan(&r.ImtID, &r.SumValuation, &r.CountValuation, &r.Rate)
		if err != nil {
			return nil, err
		}

		rates[r.ImtID] = r
	}

	return rates, rows.Err()
}

// Вставляем новую запись в БД
func (r *DbRate) Insert() error {
	conn, err := DBNew.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	query := `INSERT INTO wb.product_rates
	            (imt_id, date_rate, sum_valuation, count_valuation, rate, seller_id)
		        VALUES
						  (@imtID, @dateRate, @sumValuation, @countValuation, @rate, @sellerID)`
	args := pgx.NamedArgs{
		"imtID":          r.ImtID,
		"dateRate":       r.DateRate,
		"sumValuation":   r.SumValuation,
		"countValuation": r.CountValuation,
		"rate":           r.Rate,
		"sellerID":       r.SellerID,
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

func (pr *ProductRate) SelectProductRate() {
	conn, err := DBNew.Acquire(context.Background())
	if err != nil {
		return
	}
	defer conn.Release()

	query := `SELECT p.nm_id, p.title
	          FROM wb.products p
						WHERE p.imt_id = @imtID
						ORDER BY p.nm_id DESC
						LIMIT 1`
	args := pgx.NamedArgs{"imtID": pr.ImtID}
	row := conn.QueryRow(context.Background(), query, args)

	row.Scan(&pr.NmID, &pr.Name)
}
