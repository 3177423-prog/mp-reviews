package requests

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"auto_answer/internal/clients"
	"auto_answer/internal/consts"
	"auto_answer/internal/domain"
	"auto_answer/models"

	"github.com/imroc/req/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func OzFeedbacks() {
	defer func() {
		if r := recover(); r != nil {
			SendTgInfo(fmt.Sprint("Panic OzFeedbacks():", r))
		}
	}()

	rc := req.C()

	sellers, err := models.GetSellers(models.DBNew)
	if err != nil {
		fmt.Println("Ошибка при получении списка поставщиков:", err)
		return
	}

	var wg sync.WaitGroup
	for {
		for _, s := range *sellers {
			fmt.Printf("Селлер Озон - %s\n", s.Title)
			// На отзывы без книги не отвечаем
			if urlSheets[s.ID] == "" {
				continue
			}

			jsonToken, err := models.GetAuthenticationData(models.DBNew, s.ID, "oz")
			if err != nil {
				fmt.Printf("%s. Ошибка получения токена - %v\n", s.Title, err)
				continue
			}

			var authAPI domain.AuthenticationOzonApi
			if err := authAPI.UnmarshalJSON([]byte(jsonToken)); err != nil {
				fmt.Printf("%s. Ошибка декодирования токена - %v\n", s.Title, err)
				continue
			}

			wg.Add(1)
			go ozSellerFeedbacks(&wg, authAPI, &s, rc)
		}
		wg.Wait()

		time.Sleep(10 * time.Minute)
	}
}

func ozSellerFeedbacks(wg *sync.WaitGroup, token domain.AuthenticationOzonApi, s *models.SellerNew, rc *req.Client) {
	if wg != nil {
		defer wg.Done()
	}

	defer func() {
		if r := recover(); r != nil {
			SendTgInfo(fmt.Sprint("Panic OzFeedbacks() -> ozSellerFeedbacks():", r))
		}
	}()

	fmt.Printf("Обработка отзывов на товары Озон - %s\n", s.Title)

	ans, err := getAnswers(rc, s.ID)
	if err != nil {
		fmt.Println(s.Title, "- ошибка получения данных из таблицы Гугл!")
		return
	}

	client, err := clients.NewOzClient("https://api-seller.ozon.ru", token)
	if err != nil {
		fmt.Println("  - ошибка создания клиента:", err)
		return
	}

	messages := []errorMessage{}

	ctx := context.Background()
	dateOld := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

	// Берем только необработанные отзывы
	listRequest := domain.ReviewListRequest{
		Limit:   100,
		SortDir: consts.OptReviewListSortDirType{Value: consts.ReviewListSortDirDESC, Set: true},
		Status:  consts.OptReviewListStatusType{Value: consts.ReviewListStatusUnprocessed, Set: true},
	}
	dateBreak := false
	for {
		fmt.Printf(" - %s. OZ. Выполняем запрос для получения списка отзывов: %s\n", s.Title, listRequest.LastID.Value)
		countSend := 0
		if listResponse, err := client.ReviewList(ctx, &listRequest); err == nil {
			if len(listResponse.Reviews.Value) == 0 {
				break
			}
			for i, review := range listResponse.Reviews.Value {
				reviewDate := review.PublishedAt.Value.Format("2006-01-02")
				if i == 0 {
					fmt.Printf("   - %s. OZ. Дата отзыва %s (%t).\n", s.Title, review.PublishedAt.Value.Format("02.01.2006 15:04:05"), reviewDate < dateOld)
				}

				// Отзывы старше недели не обрабатываем
				if review.PublishedAt.Value.Before(time.Now().AddDate(0, 0, -7)) {
					fmt.Printf("   - %s. OZ. Прерываемся (дата отзыва: %s, текущая дата: %s).\n", s.Title, reviewDate, dateOld)
					dateBreak = true
					break
				}

				// На отзыв без контекста (нет текста, фото и видео) нельзя ответить
				if review.Text.Value == "" && review.PhotosAmount.Value == 0 && review.VideosAmount.Value == 0 {
					continue
				}

				// Для оценок ниже 4 отвечаем только, если отзыв пустой (только фото или видео)
				if review.Rating.Value < 4 {
					if review.Text.Value == "" {
						text, err := generateRandomPhraze(ans, review.Sku.Value, review.Rating.Value, "", "OZ")
						if err == "" {
							ozSendComment(client, ctx, text, review.ID.Value)
							countSend++
						}
					}
				} else {
					stop, isStop := isStopPhrase(ans.StopWords, review.Text.Value)
					if isStop {
						// fmt.Println(time.Now().Format("02.01.2006 15:04:05"), "OZ. Добавляем сообщение об ошибке со стоп-словом: "+strings.Join(stop, ", "))
						if !ContainsMessage(&messages, review.ID.Value) {
							messages = append(messages, errorMessage{
								Sku:        review.Sku.Value,
								Name:       "",
								Rate:       review.Rating.Value,
								NameMarket: "oz",
								FeedID:     review.ID.Value,
								ErrorName:  "Есть совпадения со стоп-словами: " + strings.Join(stop, ", "),
							})
						}
					} else {
						text, err := generateRandomPhraze(ans, review.Sku.Value, review.Rating.Value, "", "OZ")
						if err == "" {
							ozSendComment(client, ctx, text, review.ID.Value)
							countSend++
						} else {
							// fmt.Println(time.Now().Format("02.01.2006 15:04:05"), "OZ. Ошибка при генерации ответа:", err)
							if !ContainsMessage(&messages, review.ID.Value) {
								messages = append(messages, errorMessage{
									Sku:        review.Sku.Value,
									Name:       "",
									Rate:       review.Rating.Value,
									NameMarket: "oz",
									FeedID:     review.ID.Value,
									ErrorName:  err,
								})
							}
						}
					}
				}
			}

			fmt.Printf("   - %s. OZ. Отправлен комментарий на %d отзывов из %d.\n", s.Title, countSend, len(listResponse.Reviews.Value))

			if dateBreak {
				break
			}

			if listResponse.HasNext.Value {
				listRequest.LastID = listResponse.LastID
				time.Sleep(1 * time.Second)
			} else {
				break
			}
		} else {
			fmt.Printf("ошибка: %v\n", err)
			break
		}
	}

	if len(messages) > 0 {
		fmt.Println(time.Now().Format("02.01.2006 15:04:05"), s.Title, "OZ. Отправляем ошибки!")
	} else {
		fmt.Println(time.Now().Format("02.01.2006 15:04:05"), s.Title, "OZ. Нет ошибок!")
		messages = append(messages, errorMessage{NameMarket: "oz"})
	}
	sendError(rc, messages, s.ID)
}

func ozSendComment(client *clients.Client, ctx context.Context, text, reviewID string) bool {
	defer func() {
		if r := recover(); r != nil {
			SendTgInfo(fmt.Sprint("Panic OzFeedbacks() -> ozSellerFeedbacks() -> ozSendComment():", r))
		}
	}()

	request := domain.ReviewCommentCreateRequest{
		MarkReviewAsProcessed: domain.OptBool{Value: true, Set: true},
		ReviewID:              reviewID,
		Text:                  text,
	}

	if _, err := client.ReviewCommentCreate(ctx, &request); err == nil {
		time.Sleep(1 * time.Second)
	} else {
		SendTgInfo("Ошибка при выполнении запроса в <b>ozSendComment()</b>: " + err.Error())
		return false
	}

	return true
}

func OzUpdateFeedbacks() {
	defer func() {
		if r := recover(); r != nil {
			SendTgInfo(fmt.Sprint("Panic OzUpdateFeedbacks():", r))
		}
	}()

	sellers, err := models.GetSellers(models.DBNew)
	if err != nil {
		fmt.Println("OZ. Ошибка при получении списка поставщиков:", err)
		return
	}

	var wg sync.WaitGroup
	for _, s := range *sellers {
		// fmt.Printf("Продавец - %s\n", s.Title)

		jsonToken, err := models.GetAuthenticationData(models.DBNew, s.ID, "oz")
		if err != nil {
			// fmt.Printf("  ошибка получения данных для аутентификации - %v\n", err)
			continue
		}

		var authAPI domain.AuthenticationOzonApi
		if err := authAPI.UnmarshalJSON([]byte(jsonToken)); err != nil {
			// fmt.Printf("  ошибка декодирования токена Озон - %v\n", err)
			continue
		}

		wg.Add(1)
		go ozUpdateSellerFeedbacks(&wg, authAPI, &s)
	}

	wg.Wait()
}

func ozUpdateSellerFeedbacks(wg *sync.WaitGroup, token domain.AuthenticationOzonApi, s *models.SellerNew) {
	if wg != nil {
		defer wg.Done()
	}

	defer func() {
		if r := recover(); r != nil {
			SendTgInfo(fmt.Sprint("Panic OzUpdateFeedbacks() -> ozUpdateSellerFeedbacks():", r))
		}
	}()

	fmt.Printf("Обработка отзывов на товары Озон - %s\n", s.Title)

	products, err := models.GetProductIDs(models.DBPool, s.ID)
	if err != nil {
		fmt.Println("  - OZ. ошибка получения идентификаторов товаров:", err)
		return
	}

	client, err := clients.NewOzClient("https://api-seller.ozon.ru", token)
	if err != nil {
		fmt.Println("  - OZ. ошибка создания клиента:", err)
		return
	}

	ctx := context.Background()

	// Берем только необработанные отзывы
	listRequest := domain.ReviewListRequest{
		Limit:   100,
		SortDir: consts.OptReviewListSortDirType{Value: consts.ReviewListSortDirDESC, Set: true},
		Status:  consts.OptReviewListStatusType{Value: consts.ReviewListStatusProcessed, Set: true},
	}
	for {
		fmt.Printf(" - %s. OZ. Выполняем запрос для получения списка отзывов: %s\n", s.Title, listRequest.LastID.Value)
		dateBreak := false
		if listResponse, err := client.ReviewList(ctx, &listRequest); err == nil {
			if len(listResponse.Reviews.Value) == 0 {
				break
			}

			feedbacks := models.OzonFeedbacks{}
			for i, review := range listResponse.Reviews.Value {
				if i == 0 {
					fmt.Printf("   - %s. Дата отзыва %s.\n", s.Title, review.PublishedAt.Value.Format("02.01.2006 15:04:05"))
				}

				// Отзывы старше недели не обрабатываем
				if review.PublishedAt.Value.Before(time.Now().AddDate(0, 0, -7)) {
					dateBreak = true
					break
				}

				feedback := models.OzonFeedback{
					SellerID:         s.ID,
					OzonID:           review.ID.Value,
					FeedbackText:     pgtype.Text{String: review.Text.Value, Valid: review.Text.Set && review.Text.Value != ""},
					Valuation:        review.Rating.Value,
					CreatedAt:        review.PublishedAt.Value,
					ProductID:        products[review.Sku.Value],
					Sku:              review.Sku.Value,
					IsRating:         review.IsRatingParticipant.Value,
					IsOrderDelivered: review.OrderStatus.Value == consts.ReviewListOrderStatusDelivered,
				}

				if err := feedback.Get(models.DBNew); err == pgx.ErrNoRows {
					feedbacks = append(feedbacks, feedback)
				}
			}
			if errIns := feedbacks.BulkInsert(models.DBNew); errIns != nil {
				fmt.Println(time.Now().Format("02.01.2006 15:04:05"), "OZ. Ошибка вставки записей:", errIns)
			}

			if dateBreak {
				break
			}

			if listResponse.HasNext.Value {
				listRequest.LastID = listResponse.LastID
				time.Sleep(1 * time.Second)
			} else {
				break
			}
		} else {
			fmt.Printf("OZ. ошибка: %v\n", err)
			break
		}
	}
}
