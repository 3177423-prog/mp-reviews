package requests

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/imroc/req/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"auto_answer/internal/clients"
	"auto_answer/internal/consts"
	"auto_answer/internal/domain"
	"auto_answer/internal/parameters"
	"auto_answer/models"
)

func WbFeedbacks() {
	defer func() {
		if r := recover(); r != nil {
			SendTgInfo(fmt.Sprint("Panic WbFeedbacks():", r))
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
			// На отзывы без книги не отвечаем
			if urlSheets[s.ID] == "" {
				continue
			}

			jsonToken, err := models.GetAuthenticationData(models.DBNew, s.ID, "wb")
			if err != nil {
				continue
			}

			var authAPI domain.AuthenticationWildberriesApi
			if err := authAPI.UnmarshalJSON([]byte(jsonToken)); err != nil {
				continue
			}

			wg.Add(1)
			go wbSellerFeedbacks(&wg, authAPI, &s, rc)
		}
		wg.Wait()

		time.Sleep(10 * time.Minute)
	}
}

func wbSellerFeedbacks(wg *sync.WaitGroup, token domain.AuthenticationWildberriesApi, s *models.SellerNew, rc *req.Client) {
	if wg != nil {
		defer wg.Done()
	}

	defer func() {
		if r := recover(); r != nil {
			SendTgInfo(fmt.Sprint("Panic WbFeedbacks() -> wbSellerFeedbacks():", r))
		}
	}()

	fmt.Printf("Обработка отзывов на товары Wildberries - %s\n", s.Title)

	ans, err := getAnswers(rc, s.ID)
	if err != nil {
		fmt.Println(s.Title, "- ошибка получения данных из таблицы Гугл!")
		return
	}

	client, err := clients.NewWbClient("https://feedbacks-api.wildberries.ru", token)
	if err != nil {
		fmt.Println("  - ошибка создания клиента:", err)
		return
	}

	messages := []errorMessage{}

	ctx := context.Background()
	dateOld := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

	// Берем сначала необработанные отзывы, затем обработанные, но не просмотренные
	for _, typeAnswers := range []bool{false, true} {
		listParams := parameters.WbFeedbackList{
			IsAnswered: typeAnswers,
			Take:       100,
			Skip:       0,
			Order:      consts.OptWbOrderFeedbackListType{Value: consts.WbOrderFeedbackListTypeDATEDESC, Set: true},
		}
		dateBreak := false
		for {
			fmt.Printf(" - %s. WB. Выполняем запрос для получения списка отзывов: %d (обработан: %t)\n", s.Title, listParams.Skip, typeAnswers)
			countSend := 0
			if listResponse, err := client.WbFeedbackList(ctx, &listParams); err == nil {
				if len(listResponse.Data.Feedbacks.Value) == 0 {
					break
				}
				for i, review := range listResponse.Data.Feedbacks.Value {
					reviewDate := review.CreatedDate.Format("2006-01-02")
					if i == 0 {
						fmt.Printf("   - %s. WB. Дата отзыва %s (%t).\n", s.Title, review.CreatedDate.Format("02.01.2006 15:04:05"), reviewDate < dateOld)
					}

					// Отзывы старше недели не обрабатываем
					if reviewDate < dateOld {
						fmt.Printf("   - %s. WB. Прерываемся (дата отзыва: %s, текущая дата: %s).\n", s.Title, reviewDate, dateOld)
						dateBreak = true
						break
					}

					// Обработанные и просмотренные пропускаем (отзывы без текста сразу попадают в отвеченные)
					if typeAnswers && review.WasViewed {
						continue
					}

					// Для оценок ниже 4 отвечаем только, если отзыв пустой
					if review.ProductValuation < 4 {
						if review.Text.Value == "" && review.Advantages.Value == "" && review.Defects.Value == "" {
							text, err := generateRandomPhraze(ans, review.ProductDetails.NmID, review.ProductValuation, review.UserName.Value, "WB")
							if err == "" {
								wbSendAnswer(client, ctx, text, review.ID)
								countSend++
							}
						}
					} else {
						stop, isStop := isStopPhrase(ans.StopWords, review.Text.Value+" "+review.Advantages.Value+" "+review.Defects.Value)
						if isStop {
							// fmt.Println(time.Now().Format("02.01.2006 15:04:05"), "WB. Добавляем сообщение об ошибке со стоп-словом: "+strings.Join(stop, ", "))
							if !ContainsMessage(&messages, review.ID) {
								messages = append(messages, errorMessage{
									Sku:        review.ProductDetails.NmID,
									Name:       review.ProductDetails.Title,
									Rate:       review.ProductValuation,
									NameMarket: "wb",
									FeedID:     review.ID,
									ErrorName:  "Есть совпадения со стоп-словами: " + strings.Join(stop, ", "),
								})
							}
						} else {
							text, err := generateRandomPhraze(ans, review.ProductDetails.NmID, review.ProductValuation, review.UserName.Value, "WB")
							if err == "" {
								wbSendAnswer(client, ctx, text, review.ID)
								countSend++
							} else {
								// fmt.Println(time.Now().Format("02.01.2006 15:04:05"), "OZ. Ошибка при генерации ответа:", err)
								if !ContainsMessage(&messages, review.ID) {
									messages = append(messages, errorMessage{
										Sku:        review.ProductDetails.NmID,
										Name:       review.ProductDetails.Title,
										Rate:       review.ProductValuation,
										NameMarket: "wb",
										FeedID:     review.ID,
										ErrorName:  err,
									})
								}
							}
						}
					}
				}

				fmt.Printf("   - %s. WB. Отправлен комментарий на %d отзывов из %d.\n", s.Title, countSend, len(listResponse.Data.Feedbacks.Value))

				if dateBreak {
					break
				}

				listParams.Skip += listParams.Take
			} else {
				fmt.Printf("ошибка: %v\n", err)
				break
			}
		}
	}

	if len(messages) > 0 {
		fmt.Println(time.Now().Format("02.01.2006 15:04:05"), s.Title, "WB. Отправляем ошибки!")
	} else {
		fmt.Println(time.Now().Format("02.01.2006 15:04:05"), s.Title, "WB. Нет ошибок!")
		messages = append(messages, errorMessage{NameMarket: "wb"})
	}
	sendError(rc, messages, s.ID)
}

func wbSendAnswer(client *clients.Client, ctx context.Context, text, reviewID string) bool {
	defer func() {
		if r := recover(); r != nil {
			SendTgInfo(fmt.Sprint("Panic WbFeedbacks() -> wbSellerFeedbacks() -> wbSendAnswer():", r))
		}
	}()

	request := domain.WbFeedbackAnswerRequest{
		ID:   reviewID,
		Text: text,
	}

	if err := client.WbFeedbackAnswer(ctx, &request); err == nil {
		time.Sleep(1 * time.Second)
	} else {
		SendTgInfo("Ошибка при выполнении запроса в <b>wbSendAnswer()</b>: " + err.Error())
		return false
	}

	return true
}

func WbUpdateFeedbacks() {
	defer func() {
		if r := recover(); r != nil {
			SendTgInfo(fmt.Sprint("Panic WbUpdateFeedbacks():", r))
		}
	}()

	sellers, err := models.GetSellers(models.DBNew)
	if err != nil {
		fmt.Println("Ошибка при получении списка поставщиков:", err)
		return
	}

	var wg sync.WaitGroup
	for _, s := range *sellers {
		jsonToken, err := models.GetAuthenticationData(models.DBNew, s.ID, "wb")
		if err != nil {
			continue
		}

		var authAPI domain.AuthenticationWildberriesApi
		if err := authAPI.UnmarshalJSON([]byte(jsonToken)); err != nil {
			continue
		}

		wg.Add(1)
		go wbUpdateSellerFeedbacks(&wg, authAPI, &s)
	}

	wg.Wait()
}

func wbUpdateSellerFeedbacks(wg *sync.WaitGroup, token domain.AuthenticationWildberriesApi, s *models.SellerNew) {
	if wg != nil {
		defer wg.Done()
	}

	defer func() {
		if r := recover(); r != nil {
			SendTgInfo(fmt.Sprint("Panic WbUpdateFeedbacks() -> wbUpdateSellerFeedbacks():", r))
		}
	}()

	fmt.Printf("Обработка отзывов на товары Wildberries - %s\n", s.Title)

	client, err := clients.NewWbClient("https://feedbacks-api.wildberries.ru", token)
	if err != nil {
		fmt.Println("  - ошибка создания клиента:", err)
		return
	}

	ctx := context.Background()
	const MAX_SKIP uint32 = 50_000

	// Берем сначала необработанные отзывы, затем обработанные, но не просмотренные
	for _, typeAnswers := range []bool{false, true} {
		listParams := parameters.WbFeedbackList{
			IsAnswered: typeAnswers,
			Take:       1000,
			Skip:       0,
			Order:      consts.OptWbOrderFeedbackListType{Value: consts.WbOrderFeedbackListTypeDATEDESC, Set: true},
		}
		for {
			fmt.Printf(" - %s. WB. Выполняем запрос для получения списка отзывов, skip: %d\n", s.Title, listParams.Skip)
			if listResponse, err := client.WbFeedbackList(ctx, &listParams); err == nil {
				if len(listResponse.Data.Feedbacks.Value) == 0 {
					break
				}

				result := make(models.WbFeedbacks, 0, 1000)
				for _, review := range listResponse.Data.Feedbacks.Value {
					feedback := models.WbFeedback{
						SellerID:  s.ID,
						WbID:      review.ID,
						Valuation: review.ProductValuation,
						CreatedAt: review.CreatedDate,
						NmID:      review.ProductDetails.NmID,
						ImtID:     review.ProductDetails.ImtID,
					}

					if val, ok := review.ProductDetails.Article.Get(); ok {
						feedback.Article = pgtype.Text{String: val, Valid: true}
					}

					if err := feedback.Get(models.DBNew); err == pgx.ErrNoRows {
						if val, ok := review.Text.Get(); ok && len(strings.TrimSpace(val)) > 0 {
							feedback.FeedbackText = pgtype.Text{String: val, Valid: true}
						}
						if val, ok := review.Advantages.Get(); ok && len(strings.TrimSpace(val)) > 0 {
							feedback.Advantages = pgtype.Text{String: strings.TrimSpace(val), Valid: true}
						}
						if val, ok := review.Defects.Get(); ok && len(strings.TrimSpace(val)) > 0 {
							feedback.Defects = pgtype.Text{String: strings.TrimSpace(val), Valid: true}
						}
						if val, ok := review.ProductDetails.TechSize.Get(); ok && len(strings.TrimSpace(val)) > 0 && strings.TrimSpace(val) != "0" {
							feedback.TechSize = pgtype.Text{String: strings.TrimSpace(val), Valid: true}
						}
						if val, ok := review.UserName.Get(); ok && len(strings.TrimSpace(val)) > 0 {
							feedback.UserName = pgtype.Text{String: val, Valid: true}
						}
						if val, ok := review.MatchingSize.Get(); ok {
							if err := val.Validate(); err == nil && val != consts.WbFeedbackMatchingSizeTypeSPACE {
								feedback.MatchingSize = pgtype.Text{String: strings.TrimSpace(string(val)), Valid: true}
							}
						}
						if val, ok := review.ParentFeedbackID.Get(); ok && len(strings.TrimSpace(val)) > 0 {
							feedback.ParentWbID = pgtype.Text{String: strings.TrimSpace(val), Valid: true}
						}

						result = append(result, feedback)
						if len(result) >= 1000 {
							if errIns := result.BulkInsert(models.DBNew); errIns != nil {
								fmt.Println(time.Now().Format("02.01.2006 15:04:05"), "WB. Ошибка вставки записей:", errIns)
							}
							result = models.WbFeedbacks{}
							result = make(models.WbFeedbacks, 0, 1000)
						}
					}
				}

				if len(result) > 0 {
					if errIns := result.BulkInsert(models.DBNew); errIns != nil {
						fmt.Println(time.Now().Format("02.01.2006 15:04:05"), "WB. Ошибка вставки записей:", errIns)
					}
				}

				listParams.Skip += listParams.Take
				if listParams.Skip >= uint32(MAX_SKIP) {
					break
				}

				time.Sleep(5 * time.Second)
			} else {
				fmt.Printf("WB. Ошибка создания клиента: %v\n", err)
				break
			}
		}
	}
}
