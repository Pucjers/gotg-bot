package bot

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"gotg-bot/db"
	"gotg-bot/internal/file"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// The HandleUpdate function processes different commands based on user input in a Telegram bot.
func HandleUpdate(bot *tgbotapi.BotAPI, dbConn *sql.DB, update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	userState := GetUserState(userID)

	switch update.Message.Text {
	case "Add":
		SetUserState(userID, "waiting_for_voice")
		sendMessage(bot, chatID, "Send a voice message:")
	case "Edit":
		handleEditCommand(bot, dbConn, chatID, userID)
	case "Delete":
		sendMessage(bot, chatID, "Delete feature is not implemented yet.")
	case "List":
		handleListVoices(bot, dbConn, chatID, userID)
	default:
		handleFSM(bot, dbConn, update, userState)
	}
}

// The `handleFSM` function manages a finite state machine for processing user input in a Telegram bot,
// allowing users to submit voice recordings with associated metadata.
func handleFSM(bot *tgbotapi.BotAPI, dbConn *sql.DB, update tgbotapi.Update, userState string) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	switch userState {
	case "waiting_for_voice":
		if update.Message.Voice != nil {
			state := GetFSMState(userID)
			state.Voice = update.Message.Voice.FileID
			SetUserState(userID, "waiting_for_name")
			sendMessage(bot, chatID, "Name:")
		}
	case "waiting_for_name":
		state := GetFSMState(userID)
		state.Name = update.Message.Text
		SetUserState(userID, "waiting_for_description")
		sendMessage(bot, chatID, "Description:")
	case "waiting_for_description":
		state := GetFSMState(userID)
		state.Description = update.Message.Text
		SetUserState(userID, "waiting_for_tags")
		sendMessage(bot, chatID, "Tags (comma-separated):")
	case "waiting_for_tags":
		state := GetFSMState(userID)
		state.Tags = strings.Split(strings.ToLower(update.Message.Text), ", ")
		SetUserState(userID, "waiting_for_author")
		sendMessage(bot, chatID, "Author:")
	case "waiting_for_author":
		state := GetFSMState(userID)
		state.Author = update.Message.Text
		state.AuthorID = userID
		handleSaveVoice(bot, dbConn, update)
	default:
		sendMessage(bot, chatID, "Unknown command or state.")
	}
}

func handleSaveVoice(bot *tgbotapi.BotAPI, dbConn *sql.DB, update tgbotapi.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	state := GetFSMState(userID)

	voicePath, err := file.DownloadVoiceFile(bot, state.Voice)
	if err != nil {
		log.Printf("Error downloading voice file: %v", err)
		sendMessage(bot, chatID, "Error downloading your voice file. Please try again.")
		return
	}

	err = db.SaveVoiceToDB(dbConn, voicePath, state.Name, state.Description, state.Tags, state.Author, state.AuthorID)
	if err != nil {
		log.Printf("Error saving data to DB: %v", err)
		sendMessage(bot, chatID, "Error saving your data. Please try again.")
		return
	}

	sendMessage(bot, chatID, "Voice saved successfully!")
	SetUserState(userID, "")
}

func handleEditCommand(bot *tgbotapi.BotAPI, dbConn *sql.DB, chatID, userID int64) {
	voices, err := db.GetUserVoices(dbConn, userID)
	if err != nil {
		log.Printf("Error retrieving voices: %v", err)
		sendMessage(bot, chatID, "Error retrieving your recordings.")
		return
	}

	if len(voices) == 0 {
		sendMessage(bot, chatID, "You have no recordings to edit.")
		return
	}

	var response strings.Builder
	for i, voice := range voices {
		response.WriteString(fmt.Sprintf("%d. Name: %s\nDescription: %s\n", i+1, voice.Name, voice.Description))
	}

	response.WriteString("\nEnter the number of the recording you want to edit:")
	sendMessage(bot, chatID, response.String())
	SetUserState(userID, "waiting_for_edit_selection")
}

// The function `handleListVoices` retrieves a user's recorded voices from a database and sends them as
// a message to a chat using a Telegram bot.
func handleListVoices(bot *tgbotapi.BotAPI, dbConn *sql.DB, chatID, userID int64) {
	voices, err := db.GetUserVoices(dbConn, userID)
	if err != nil {
		log.Printf("Error retrieving voices: %v", err)
		sendMessage(bot, chatID, "Error retrieving your recordings.")
		return
	}

	if len(voices) == 0 {
		sendMessage(bot, chatID, "You have no recordings.")
		return
	}

	var response strings.Builder
	for _, voice := range voices {
		response.WriteString(fmt.Sprintf("Name: %s\nDescription: %s\n\n", voice.Name, voice.Description))
	}

	sendMessage(bot, chatID, response.String())
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}
