package main

import (
	"auto_answer/models"
	"auto_answer/requests"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/robfig/cron/v3"
)

func main() {
	models.ConnectOldDB()
	models.ConnectDB()

	// requests.CheckRating()
	// return

	scheduler := cron.New()
	defer scheduler.Stop()

	scheduler.AddFunc("20 7,23 * * *", requests.WbUpdateFeedbacks) // Сохраняем все отзывы в БД
	scheduler.AddFunc("30 7,23 * * *", requests.OzUpdateFeedbacks) // Сохраняем все отзывы Озон в БД
	// scheduler.AddFunc("0 9 * * *", requests.CheckRating)           // Проверяем пересечение нижнего порога по оценке

	go scheduler.Start()

	go requests.OzFeedbacks()
	go requests.WbFeedbacks()

	requests.SendTgInfo("Стартовал!")

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Logger.Fatal(e.Start(":9005"))
}
