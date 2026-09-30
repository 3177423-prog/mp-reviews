package models

import (
	"context"
	"errors"
	"net/http"
)

type OzCookies struct {
	SupplierID int
	Key        string // Ключ cookie
	Value      string // Значение ключа
}

func SelectOzonCookies(sellerID int16) ([]*http.Cookie, error) {
	ozCookies := []*http.Cookie{}
	conn, err := DBPool.Acquire(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	rows, _ := conn.Query(context.Background(),
		`SELECT key, value FROM oz_cookies WHERE supplier_id = $1`, sellerID)
	for rows.Next() {
		var cookie OzCookies
		err = rows.Scan(&cookie.Key, &cookie.Value)
		if err != nil {
			return nil, err
		}

		cookieToken := &http.Cookie{
			Name:   cookie.Key,
			Value:  cookie.Value,
			Path:   "/",
			Domain: ".ozon.ru",
		}
		ozCookies = append(ozCookies, cookieToken)
	}

	return ozCookies, nil
}

func UpdateOzonCookies(sellerID int16, cookies []*http.Cookie) error {
	conn, err := DBPool.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	for _, c := range cookies {
		ct, err := conn.Exec(context.Background(),
			`UPDATE oz_cookies SET value = $3 WHERE supplier_id = $1 AND key = $2`,
			sellerID, c.Name, c.Value)
		if err != nil {
			return err
		}

		if ct.RowsAffected() == 0 {
			ct, err := conn.Exec(context.Background(),
				`INSERT INTO oz_cookies (supplier_id, key, value) VALUES ($1, $2, $3)`,
				sellerID, c.Name, c.Value)
			if err != nil {
				return err
			}

			if ct.RowsAffected() == 0 {
				return errors.New("ошибка при обновлении/добавлении записи")
			}

		}
	}

	return nil
}
