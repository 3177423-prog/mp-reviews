package models

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DBPool *pgxpool.Pool
var DBNew *pgxpool.Pool

func ConnectOldDB() {
	dbURL := "host= port= dbname= user= password= sslmode=disable"
	pool, err := pgxpool.New(context.Background(), dbURL)

	if err != nil {
		log.Fatal(err)
		panic("Не удалось подключиться к базе данных!")
	}

	DBPool = pool
}

func ConnectDB() {
	dbURL := "host= port= dbname= user= password= sslmode=disable"
	pool, err := pgxpool.New(context.Background(), dbURL)

	if err != nil {
		log.Fatal(err)
		panic("Не удалось подключиться к базе данных!")
	}

	DBNew = pool
}
