package requests

import (
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/imroc/req/v3"
)

var urlSheets []string = []string{
	"",
	"https://script.google.com/macros/s/APPS_SCRIPT_ID_REDACTED_1/exec", // Маркетспейс
	"https://script.google.com/macros/s/APPS_SCRIPT_ID_REDACTED_2/exec",   // Цифровой Ритейл
	"https://script.google.com/macros/s/APPS_SCRIPT_ID_REDACTED_3/exec",   // Балт Трейд
	"https://script.google.com/macros/s/APPS_SCRIPT_ID_REDACTED_4/exec",   // Еком Дизайн
	"", // Alveria - не торгует на Озон
	"https://script.google.com/macros/s/APPS_SCRIPT_ID_REDACTED_5/exec", // Центр Бытовой техники и Электроники
	"", // Маркетспейс (оферта) - пока нет книги
	"https://script.google.com/macros/s/APPS_SCRIPT_ID_REDACTED_6/exec", // Хоум Брендс
}

type (
	recomendation struct {
		NameProducts  []string // Наименования продукта (разные)
		Recomendation string   // Текст рекомендации
		Sku           int64    // Идентификатор товара
	}

	shablonAnswer struct {
		Greetings   []string // Приветствия
		Thanks      []string // Благодарности за отзыв
		Motivations []string // Мотивации
		Mains       []string // Основные тексты
		RecomOZ     []string // Рекомендации Ozon
		RecomWB     []string // Рекомендации Wildberries
		Partings    []string // Прощания
	}

	answers struct {
		Star45      shablonAnswer           // Кусочки шаблона для ответов на 4-5 звезд
		Star13      shablonAnswer           // Кусочки шаблона для ответов на 1-3 звезд
		StopWords   []string                // Стоп-слова
		RecomOZList map[int64]recomendation // Список рекомендаций по конткретным товарам Ozon
		RecomWBList map[int64]recomendation // Список рекомендаций по конткретным товарам Wildberries
	}

	errorMessage struct {
		Sku        int64  `json:"sku"`     // Идентификатор товара
		Name       string `json:"name"`    // Наименование товара
		Rate       uint8  `json:"rate"`    // Оценка
		NameMarket string `json:"mm"`      // Код площадки: wb, oz, sb, ya
		FeedID     string `json:"feed_id"` // Идентификатор отзыва
		ErrorName  string `json:"error"`   // Описание ошибки
	}
)

// Получаем данные для формирования ответов из таблицы
func getAnswers(client *req.Client, sellerID int16) (answers, error) {
	type tempAnswers struct {
		Star45 struct {
			Greetings   []string `json:"greetings"`   // Приветствия
			Thanks      []string `json:"thanks"`      // Благодарности за отзыв
			Motivations []string `json:"motivations"` // Мотивации
			Mains       []string `json:"mains"`       // Основные тексты
			RecomOZ     []string `json:"recom_oz"`    // Рекомендации Ozon
			RecomWB     []string `json:"recom_wb"`    // Рекомендации Wildberries
			Partings    []string `json:"partings"`    // Прощания
		} `json:"star_45"`
		Star13 struct {
			Greetings   []string `json:"greetings"`   // Приветствия
			Thanks      []string `json:"thanks"`      // Благодарности за отзыв
			Motivations []string `json:"motivations"` // Мотивации
			Mains       []string `json:"mains"`       // Основные тексты
			RecomOZ     []string `json:"recom_oz"`    // Рекомендации Ozon
			RecomWB     []string `json:"recom_wb"`    // Рекомендации Wildberries
			Partings    []string `json:"partings"`    // Прощания
		} `json:"star_13"`
		StopWords   []string   `json:"stop_words"` // Стоп-слова
		RecomOZList []struct { // Список рекомендаций по конткретным товарам Ozon
			NameProducts  []string `json:"name_products"` // Наименования продукта (разные)
			Recomendation string   `json:"recomendation"` // Текст рекомендации
			Sku           int64    `json:"sku"`           // Идентификатор товара
		} `json:"recom_oz_list"`
		RecomWBList []struct { // Список рекомендаций по конткретным товарам Wildberries
			NameProducts  []string `json:"name_products"` // Наименования продукта (разные)
			Recomendation string   `json:"recomendation"` // Текст рекомендации
			Sku           int64    `json:"sku"`           // Идентификатор товара
		} `json:"recom_wb_list"`
	}

	var ans answers

	resp, err := client.R().Get(urlSheets[sellerID])

	if err != nil {
		SendTgInfo("Ошибка при выполнении запроса в <b>getAnswers()</b>: " + err.Error())
		return ans, err
	}

	var tans tempAnswers
	if resp.StatusCode == http.StatusOK {
		err = resp.UnmarshalJson(&tans)
		if err != nil {
			SendTgInfo("Ошибка при выполнении UnmarshalJson в <b>getAnswers()</b>: " + err.Error())
		}
	}

	star45 := shablonAnswer{
		Greetings:   tans.Star45.Greetings,
		Thanks:      tans.Star45.Thanks,
		Motivations: tans.Star45.Motivations,
		Mains:       tans.Star45.Mains,
		RecomOZ:     tans.Star45.RecomOZ,
		RecomWB:     tans.Star45.RecomWB,
		Partings:    tans.Star45.Partings,
	}
	star13 := shablonAnswer{
		Greetings:   tans.Star13.Greetings,
		Thanks:      tans.Star13.Thanks,
		Motivations: tans.Star13.Motivations,
		Mains:       tans.Star13.Mains,
		RecomOZ:     tans.Star13.RecomOZ,
		RecomWB:     tans.Star13.RecomWB,
		Partings:    tans.Star13.Partings,
	}
	ans = answers{
		Star45:      star45,
		Star13:      star13,
		StopWords:   tans.StopWords,
		RecomOZList: map[int64]recomendation{},
		RecomWBList: map[int64]recomendation{},
	}
	for _, r := range tans.RecomOZList {
		ans.RecomOZList[r.Sku] = recomendation{NameProducts: r.NameProducts, Recomendation: r.Recomendation, Sku: r.Sku}
	}
	for _, r := range tans.RecomWBList {
		ans.RecomWBList[r.Sku] = recomendation{NameProducts: r.NameProducts, Recomendation: r.Recomendation, Sku: r.Sku}
	}

	return ans, nil
}

// Записываем в таблицу рекомендаций SKU, которого там ранее не было
func sendError(client *req.Client, messages []errorMessage, supplierID int16) {
	type payload struct {
		Messages []errorMessage `json:"messages"`
	}

	// fmt.Printf("  - %d - отправляем ошибки в гугл: %d\n", supplierID, len(messages))

	pl := payload{Messages: messages}

	_, err := client.R().SetBody(pl).Post(urlSheets[supplierID])

	if err != nil {
		SendTgInfo("Ошибка при выполнении запроса в <b>sendError()</b>: " + err.Error())
	}

	// fmt.Printf("  - результат отправки: %s\n", resp.Status)
}

// Генерируем случайный ответ из набора блоков
func generateRandomPhraze(ans answers, sku int64, valuation uint8, name, mm string) (string, string) {
	s := rand.NewSource(time.Now().Unix())
	r := rand.New(s)

	var rnd int
	var greeting, thank, motivation, main, parting, nameProduct, recomendation, recomendationText string

	// В зависимости от оценки идем по разным веткам формирования ответа
	switch valuation {
	case 1, 2, 3:
		// Приветствие
		rnd = r.Intn(len(ans.Star13.Greetings))
		greeting = ans.Star13.Greetings[rnd]

		// Благодарность за отзыв
		rnd = r.Intn(len(ans.Star13.Thanks))
		thank = ans.Star13.Thanks[rnd]

		// Мотивация
		rnd = r.Intn(len(ans.Star13.Motivations))
		motivation = ans.Star13.Motivations[rnd]

		// Основное тело ответа
		rnd = r.Intn(len(ans.Star13.Mains))
		main = ans.Star13.Mains[rnd]

		switch mm {
		case "OZ":
			// Рекомендации
			rnd = r.Intn(len(ans.Star13.RecomOZ))
			recomendation = ans.Star13.RecomOZ[rnd]

			recs, seek := ans.RecomOZList[sku]
			if seek {
				// Имя продукта и текст рекомендации
				rnd = r.Intn(len(recs.NameProducts))
				nameProduct = recs.NameProducts[rnd]
				recomendationText = recs.Recomendation
			} else {
				return "", "Нет рекомендаций!"
			}
		case "WB":
			// Рекомендации
			rnd = r.Intn(len(ans.Star13.RecomWB))
			recomendation = ans.Star13.RecomWB[rnd]

			recs, seek := ans.RecomWBList[sku]
			if seek {
				// Имя продукта и текст рекомендации
				rnd = r.Intn(len(recs.NameProducts))
				nameProduct = recs.NameProducts[rnd]
				recomendationText = recs.Recomendation
			} else {
				return "", "Нет рекомендаций!"
			}
		}

		// Прощание
		rnd = r.Intn(len(ans.Star13.Partings))
		parting = ans.Star13.Partings[rnd]
	case 4, 5:
		// Приветствие
		rnd = r.Intn(len(ans.Star45.Greetings))
		greeting = ans.Star45.Greetings[rnd]

		// Благодарность за отзыв
		rnd = r.Intn(len(ans.Star45.Thanks))
		thank = ans.Star45.Thanks[rnd]

		// Мотивация
		rnd = r.Intn(len(ans.Star45.Motivations))
		motivation = ans.Star45.Motivations[rnd]

		// Основное тело ответа
		rnd = r.Intn(len(ans.Star45.Mains))
		main = ans.Star45.Mains[rnd]

		switch mm {
		case "OZ":
			// Рекомендации
			rnd = r.Intn(len(ans.Star45.RecomOZ))
			recomendation = ans.Star45.RecomOZ[rnd]

			recs, seek := ans.RecomOZList[sku]
			if seek {
				// Имя продукта и текст рекомендации
				rnd = r.Intn(len(recs.NameProducts))
				nameProduct = recs.NameProducts[rnd]
				recomendationText = recs.Recomendation
			} else {
				return "", "Нет рекомендаций!"
			}
		case "WB":
			// Рекомендации
			rnd = r.Intn(len(ans.Star45.RecomWB))
			recomendation = ans.Star45.RecomWB[rnd]

			recs, seek := ans.RecomWBList[sku]
			if seek {
				// Имя продукта и текст рекомендации
				rnd = r.Intn(len(recs.NameProducts))
				nameProduct = recs.NameProducts[rnd]
				recomendationText = recs.Recomendation
			} else {
				return "", "Нет рекомендаций!"
			}
		}

		// Прощание
		rnd = r.Intn(len(ans.Star45.Partings))
		parting = ans.Star45.Partings[rnd]
	}

	// Собираем полный текст отзыва
	text := greeting + "\n" + thank + "\n" + motivation + "\n" + main + "\n" + recomendation + "\n" + parting

	// Заменяем подстановочные
	text = strings.ReplaceAll(text, "{{productName}}", nameProduct)
	text = strings.ReplaceAll(text, "{{recomendation}}", recomendationText)
	if len(name) == 0 {
		text = strings.ReplaceAll(text, ", {{name}}", "")
		text = strings.ReplaceAll(text, "{{name}},", "")
	} else {
		text = strings.ReplaceAll(text, "{{name}}", name)
	}

	return text, ""
}

// Возвращаем псевдослучайное число в диапазоне
// func randInt(min, max int) int {
// 	s := rand.NewSource(time.Now().Unix())
// 	r := rand.New(s)
// 	return r.Intn(max-min+1) + min
// }

// Проверяем есть ли стоп-слова в отзыве
// func isStopWord(stopWords []string, text string) ([]string, bool) {
// 	re := regexp.MustCompile(`[а-яё0-9a-z]+`)
// 	words := re.FindAllString(strings.ToLower(text), -1)
// 	sovp := []string{}
// 	isStop := false
// 	for _, sword := range stopWords {
// 		sw := strings.TrimSpace(strings.ToLower(sword))
// 		start := false
// 		finish := false
// 		if strings.HasPrefix(sw, "*") {
// 			start = true
// 		}
// 		if strings.HasSuffix(sw, "*") {
// 			finish = true
// 		}
// 		sw = strings.ReplaceAll(sw, "*", "")
// 		if start && finish { // Если звездочки с обоих сторон
// 			for _, w := range words {
// 				if strings.Contains(w, sw) {
// 					isStop = true
// 					sovp = append(sovp, sword)
// 					break
// 				}
// 			}
// 		} else if start { // Если звездочка только в начале
// 			for _, w := range words {
// 				if strings.HasSuffix(w, sw) {
// 					isStop = true
// 					sovp = append(sovp, sword)
// 					break
// 				}
// 			}
// 		} else if finish { // Если звездочка только в конце
// 			for _, w := range words {
// 				if strings.HasPrefix(w, sw) {
// 					isStop = true
// 					sovp = append(sovp, sword)
// 					break
// 				}
// 			}
// 		} else { // Если нет звездочек
// 			for _, w := range words {
// 				if w == sw {
// 					isStop = true
// 					sovp = append(sovp, sword)
// 					break
// 				}
// 			}
// 		}
// 	}
// 	return sovp, isStop
// }

func isStopPhrase(stopWords []string, text string) ([]string, bool) {
	re := regexp.MustCompile(`[а-яё0-9a-z]+`)
	rea := regexp.MustCompile(`[а-яё0-9a-z*]+`)
	words := re.FindAllString(strings.ToLower(text), -1)

	sovp := []string{}
	isStop := false

	for _, sword := range stopWords {
		sws := rea.FindAllString(strings.ToLower(sword), -1)
		phraseMatches := make([][]int, len(sws))

		for i, sw := range sws {
			start := false
			finish := false
			if strings.HasPrefix(sw, "*") {
				start = true
			}
			if strings.HasSuffix(sw, "*") {
				finish = true
			}

			sw = strings.ReplaceAll(sw, "*", "")

			if start && finish { // Если звездочки с обоих сторон
				for j, w := range words {
					if strings.Contains(w, sw) {
						if i == 0 {
							phraseMatches[i] = append(phraseMatches[i], j)
						} else {
							for _, p := range phraseMatches[i-1] {
								if p == j-1 {
									phraseMatches[i] = append(phraseMatches[i], j)
								}
							}
						}
					}
				}
			} else if start { // Если звездочка только в начале
				for j, w := range words {
					if strings.HasSuffix(w, sw) {
						if i == 0 {
							phraseMatches[i] = append(phraseMatches[i], j)
						} else {
							for _, p := range phraseMatches[i-1] {
								if p == j-1 {
									phraseMatches[i] = append(phraseMatches[i], j)
								}
							}
						}
					}
				}
			} else if finish { // Если звездочка только в конце
				for j, w := range words {
					if strings.HasPrefix(w, sw) {
						if i == 0 {
							phraseMatches[i] = append(phraseMatches[i], j)
						} else {
							for _, p := range phraseMatches[i-1] {
								if p == j-1 {
									phraseMatches[i] = append(phraseMatches[i], j)
								}
							}
						}
					}
				}
			} else { // Если нет звездочек
				for j, w := range words {
					if w == sw {
						if i == 0 {
							phraseMatches[i] = append(phraseMatches[i], j)
						} else {
							for _, p := range phraseMatches[i-1] {
								if p == j-1 {
									phraseMatches[i] = append(phraseMatches[i], j)
								}
							}
						}
					}
				}
			}
		}

		if len(phraseMatches[len(phraseMatches)-1]) > 0 {
			isStop = true
			sovp = append(sovp, sword)
		}
	}

	return sovp, isStop
}

func ContainsMessage(arr *[]errorMessage, uuid string) bool {
	for _, n := range *arr {
		if uuid == n.FeedID {
			return true
		}
	}
	return false
}
