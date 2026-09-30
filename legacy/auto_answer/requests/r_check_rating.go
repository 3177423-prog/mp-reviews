package requests

// import (
// 	"fmt"
// 	"math"
// 	"net/http"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/imroc/req/v3"

// 	"auto_answer/models"
// )

// const (
// 	THRESHOLD float64 = 4.7
// )

// // Проверяем снижение рейтинга ниже порога
// func CheckRating() {
// 	sellers, err := models.SelectAllSellers()
// 	if err != nil {
// 		fmt.Println("Ошибка получения списка поставщиков:", err)
// 		return
// 	}

// 	cd := time.Now()
// 	currentDate := time.Date(cd.Year(), cd.Month(), cd.Day(), 0, 0, 0, 0, time.Local)
// 	yesterday := currentDate.AddDate(0, 0, -1)

// 	client := req.C()

// 	thresholdStr := strconv.FormatFloat(THRESHOLD, 'f', 1, 64)

// 	for _, s := range sellers {
// 		fmt.Println()
// 		fmt.Println("Поставщик:", s.Name)
// 		products, _ := models.SelectAllImts(s.ID)
// 		fmt.Println("  - всего товаров:", len(products))
// 		if len(products) == 0 {
// 			continue
// 		}

// 		currentRaitings := getAllRatings(client, currentDate, s.ID, &products)

// 		yesterdayRatings, err := models.SelectAllBDRatings(THRESHOLD, s.ID, yesterday)
// 		if err != nil {
// 			fmt.Println("Ошибка получения средней оценки по отзывам за вчера:", err)
// 		}

// 		for _, r := range currentRaitings {
// 			if ro, seek := yesterdayRatings[r.ImtID]; seek {
// 				r.SelectProductRate()

// 				url := generateWbCardURL(r.NmID)
// 				curRate := strconv.FormatFloat(r.Rate, 'f', 1, 64)
// 				oldRate := strconv.FormatFloat(ro.Rate, 'f', 1, 64)

// 				text := fmt.Sprintf("<b>*** %s *** ВНИМАНИЕ!!!\nСнижение оценки ниже заданного порога в %s!</b>\n\n<i>ОЦЕНКИ: <b>сегодня - %s</b>, вчера - %s, оценок - %d</i>\n\n- <a href=\"%s\">%s</a>",
// 					s.Name, thresholdStr, curRate, oldRate, r.CountValuation, url, r.Name)

// 				SendTgChatRate(text)
// 				time.Sleep(time.Millisecond * 500)
// 			}
// 		}
// 	}
// }

// func getAllRatings(client *req.Client, currentDate time.Time, sellerID int16, ids *models.MapProductRate) models.MapProductRate {
// 	type WbCard struct {
// 		Products []struct {
// 			NmID           int64   `json:"id"`
// 			ImtID          int64   `json:"root"`
// 			ReviewRating   float64 `json:"reviewRating"`
// 			NmReviewRating float64 `json:"nmReviewRating"`
// 			Feedbacks      int32   `json:"feedbacks"`
// 			NmFeedbacks    int32   `json:"nmFeedbacks"`
// 		} `json:"products"`
// 	}

// 	result := models.MapProductRate{}

// 	allIds := []int64{}
// 	for _, pr := range *ids {
// 		allIds = append(allIds, pr.NmID)
// 	}

// 	limit := 50
// 	countPage := len(allIds)/limit + 1

// 	for page := 0; page < countPage; page++ {
// 		fmt.Println("  - обрабатываем страницу:", page+1)

// 		start := page * limit
// 		finish := page*limit + limit
// 		if len(allIds)-page*limit < limit {
// 			finish = len(allIds)
// 		}
// 		nmIds := []string{}

// 		for i := start; i < finish; i++ {
// 			nmIds = append(nmIds, strconv.FormatInt(allIds[i], 10))
// 		}

// 		resp, err := client.R().
// 			AddQueryParam("appType", "1").
// 			AddQueryParam("curr", "rub").
// 			AddQueryParam("spp", "30").
// 			AddQueryParam("hide_dtype", "11").
// 			AddQueryParam("dest", "-1257484").
// 			AddQueryParam("ab_testing", "false").
// 			AddQueryParam("lang", "ru").
// 			AddQueryParam("nm", strings.Join(nmIds, ";")).
// 			Get("https://card.wb.ru/cards/v4/detail")

// 		if err != nil {
// 			SendTgInfo("Ошибка при выполнении запроса в <b>getRateFromImtID()</b>: " + err.Error())
// 			continue
// 		}

// 		if resp.StatusCode == http.StatusOK {
// 			var card WbCard
// 			err := resp.UnmarshalJson(&card)
// 			if err != nil {
// 				SendTgInfo("Ошибка при выполнении UnmarshalJson в <b>getRateFromImtID()</b>: " + err.Error())
// 				continue
// 			}

// 			for _, p := range card.Products {
// 				dbr := models.DbRate{
// 					ImtID:          p.ImtID,
// 					DateRate:       currentDate,
// 					SumValuation:   int32(math.Round(p.ReviewRating * float64(p.Feedbacks))),
// 					CountValuation: p.Feedbacks,
// 					Rate:           p.ReviewRating,
// 					SellerID:       sellerID,
// 				}
// 				if err := dbr.Insert(); err != nil {
// 					fmt.Println("  - ошибка записи рейтинга:", err)
// 				}

// 				if p.ReviewRating < THRESHOLD && p.ReviewRating > 0.0 {
// 					pr := (*ids)[p.ImtID]
// 					pr.Rate = p.ReviewRating
// 					pr.CountValuation = p.Feedbacks

// 					result[p.ImtID] = pr
// 				}
// 			}
// 		} else {
// 			fmt.Println("Вернулся ошибочный код при выполнении запроса getRateFromImtID():", resp.StatusCode)
// 		}
// 	}

// 	return result
// }

// func generateWbCardURL(nmID int64) string {
// 	return fmt.Sprintf("https://www.wildberries.ru/catalog/%d/detail.aspx?targetUrl=GP", nmID)
// }
