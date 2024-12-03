package bot

import (
	"database/sql"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var Bot *tgbotapi.BotAPI

// The `Run` function initializes a Telegram bot using a provided API token and continuously listens
// for updates to handle using the `HandleUpdate` function.
func Run(db *sql.DB) {
	token := os.Getenv("TELEGRAM_APITOKEN")

	var err error
	Bot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal("Failed to create bot:", err)
	}

	Bot.Debug = true
	log.Printf("Authorized on account %s", Bot.Self.UserName)

	updateConfig := tgbotapi.NewUpdate(0)
	updates := Bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		HandleUpdate(Bot, db, update)
	}
}
