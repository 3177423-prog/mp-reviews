package requests

import (
	"github.com/imroc/req/v3"
)

// Отправляем информационное сообщение в мой чат
func SendTgInfo(message string) {
	text := "Сервис <b>auto_answer</b>.\n" + message
	var chatID int64 = 0 // REDACTED
	sendTgBotMessage(text, chatID, true)
}

// Отправляем информационное сообщение в мой чат
func SendTgChatRate(message string) {
	// var chatID int64 = 0 // REDACTED
	var chatID int64 = 0 // REDACTED // Снижение рейтинга бот (новый с 13.11.2024)
	sendTgBotMessage(message, chatID, true)
}

// Отправляем сообщение в чат по его ID
func sendTgBotMessage(text string, chatID int64, previewLinksDisabled bool) {
	type LinkPreviewOptions struct {
		IsDisabled bool `json:"is_disabled"`
	}

	type body struct {
		ChatID             int64               `json:"chat_id"`
		ParseMode          string              `json:"parse_mode"`
		Text               string              `json:"text"`
		LinkPreviewOptions *LinkPreviewOptions `json:"link_preview_options,omitempty"`
	}

	client := req.C() // .DevMode()

	b := body{
		ChatID:    chatID,
		ParseMode: "HTML",
		Text:      text,
	}

	if previewLinksDisabled {
		previewOptions := LinkPreviewOptions{
			IsDisabled: true,
		}

		b.LinkPreviewOptions = &previewOptions
	}

	client.R().
		SetBodyJsonMarshal(b).
		Post("https://api.telegram.org/botTELEGRAM_BOT_TOKEN_REDACTED/sendMessage")
}
