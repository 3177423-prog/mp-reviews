package models

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OzonIDs map[int64]int64

func GetProductIDs(db *pgxpool.Pool, sellerID int16) (OzonIDs, error) {
	result := OzonIDs{}

	conn, err := db.Acquire(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	query := `SELECT p.product_id, p.sku
	          FROM oz_products p
						WHERE p.product_id IS NOT NULL AND p.sku IS NOT NULL AND p.supplier_id = @sellerID`
	args := pgx.NamedArgs{"sellerID": sellerID}
	rows, _ := conn.Query(context.Background(), query, args)
	for rows.Next() {
		var productID, sku int64
		err = rows.Scan(&productID, &sku)
		if err != nil {
			return nil, err
		}

		result[sku] = productID
	}

	return result, rows.Err()

}
