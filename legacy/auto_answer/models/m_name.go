package models

import (
	"context"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Name struct {
	NameL string
}

func GetName(fullName string) string {
	fullName = strings.TrimSpace(fullName)
	if len(fullName) == 0 {
		return ""
	}
	if fullName == "Пользователь предпочёл скрыть свои данные" {
		return ""
	}

	conn, err := DBPool.Acquire(context.Background())
	if err != nil {
		return ""
	}
	defer conn.Release()

	seekNames := strings.Split(strings.ToLower(fullName), " ")

	var name = Name{}
	for _, w := range seekNames {
		row := conn.QueryRow(context.Background(), `SELECT name_l FROM names WHERE name_l = $1`, w)

		err = row.Scan(&name.NameL)
		if err != nil {
			continue
		}
		return cases.Title(language.Russian).String(name.NameL)
	}
	return ""
}
