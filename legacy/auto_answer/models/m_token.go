package models

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Получаем информацию о токенах, ключах доступа и т.д. по ID продавца и коду маркетплейса
func GetAuthenticationData(db *pgxpool.Pool, sellerID int16, marketplaceCode string) (string, error) {
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return "", err
	}
	defer conn.Release()

	query := `SELECT mt.data_tokens 
		        FROM public.marketplace_tokens mt, public.marketplaces m, public.sellers s
						WHERE s.id = mt.seller_id AND s.deleted_at IS NULL AND m.id = mt.marketplace_id AND s.id = @sellerID AND m.code = @code`
	args := pgx.NamedArgs{"sellerID": sellerID, "code": marketplaceCode}
	row := conn.QueryRow(context.Background(), query, args)

	var result string
	err = row.Scan(&result)
	if err != nil {
		return "", err
	}

	return result, nil
}
