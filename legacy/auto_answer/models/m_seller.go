package models

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Seller struct {
	ID           int16      // Идентификатор
	CreatedAt    time.Time  // Дата создания
	UpdatedAt    time.Time  // Дата обновления
	DeletedAt    *time.Time // Дата удаления
	Name         string     // Наименование поставщика
	WbSupplierId *string    // Идентификатор поставщика в WB (внутреннее API)
	WbToken      *string    // Токен WB для доступа к внутреннему API
	WbWildToken  *string    // Личный токен пользователя (для смены WBToken)
	WbBaseToken  *string    // Токен для доступа к базовому API
	OzonClientId *string
	OzonApiKey   *string
}

// Продавец товаров
type SellerNew struct {
	ID        int16      // Идентификатор
	Title     string     // Наименование продавца
	CreatedAt time.Time  // Дата создания
	UpdatedAt time.Time  // Дата обновления
	DeletedAt *time.Time // Дата удаления
	IsDeleted bool       // Признак удаления продавца
}
type Sellers []SellerNew

// Получаем всех поставщиков
func SelectAllSellers() ([]*Seller, error) {
	suppliers := []*Seller{}
	conn, err := DBPool.Acquire(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	rows, _ := conn.Query(context.Background(),
		`SELECT id, created_at, updated_at, deleted_at, name, wb_supplier_id, wb_token, wb_wild_token, wb_api_token, ozon_client_id,
			      ozon_api_key FROM suppliers WHERE deleted_at IS NULL ORDER BY id`)
	for rows.Next() {
		var supplier Seller
		err = rows.Scan(&supplier.ID, &supplier.CreatedAt, &supplier.UpdatedAt, &supplier.DeletedAt, &supplier.Name, &supplier.WbSupplierId,
			&supplier.WbToken, &supplier.WbWildToken, &supplier.WbBaseToken, &supplier.OzonClientId, &supplier.OzonApiKey)
		if err != nil {
			return nil, err
		}

		suppliers = append(suppliers, &supplier)
	}

	return suppliers, rows.Err()
}

// Получаем информацию о поставщике по его ID
func (seller *Seller) Select() error {
	conn, err := DBPool.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	row := conn.QueryRow(context.Background(),
		`SELECT created_at, updated_at, deleted_at, name, wb_supplier_id, wb_token, wb_wild_token, wb_api_token, ozon_client_id,
			      ozon_api_key
		 FROM suppliers WHERE id = $1`, seller.ID)

	err = row.Scan(&seller.CreatedAt, &seller.UpdatedAt, &seller.DeletedAt, &seller.Name, &seller.WbSupplierId,
		&seller.WbToken, &seller.WbWildToken, &seller.WbBaseToken, &seller.OzonClientId, &seller.OzonApiKey)
	if err != nil {
		return err
	}

	return nil
}

// Получаем информацию о поставщике по его ID
func (seller *Seller) Get() error {
	conn, err := DBNew.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	query := `SELECT s.created_at, s.updated_at, s.deleted_at, s.title, mt.access_token 
		        FROM public.sellers s, public.marketplace_tokens mt
						WHERE s.id = mt.seller_id AND s.id = @sellerID AND mt.marketplace_id = 1`
	args := pgx.NamedArgs{"sellerID": seller.ID}
	row := conn.QueryRow(context.Background(), query, args)

	err = row.Scan(&seller.CreatedAt, &seller.UpdatedAt, &seller.DeletedAt, &seller.Name, &seller.WbBaseToken)
	if err != nil {
		return err
	}

	return nil
}

// Получаем всех продавцов
func GetSellers(db *pgxpool.Pool) (*Sellers, error) {
	sellers := Sellers{}
	conn, err := db.Acquire(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	query := `SELECT id, title, is_deleted, created_at, updated_at, deleted_at
            FROM public.sellers
						ORDER BY id`
	rows, _ := conn.Query(context.Background(), query)
	for rows.Next() {
		var s SellerNew
		err = rows.Scan(&s.ID, &s.Title, &s.IsDeleted, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt)
		if err != nil {
			return nil, err
		}

		sellers = append(sellers, s)
	}

	return &sellers, rows.Err()
}
