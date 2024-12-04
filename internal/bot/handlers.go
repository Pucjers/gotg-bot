package bot

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
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

	if update.Message.IsCommand() {
		handleCommand(bot, update)
		return
	}

	if userState != "" {
		handleFSM(bot, dbConn, update, userState)
		return
	}

	switch update.Message.Text {
	case "Add":
		SetUserState(userID, "waiting_for_voice")
		sendMessage(bot, chatID, "Send a voice message:")
	case "Edit":
		handleEditCommand(bot, dbConn, update)
	case "Delete":
		handleDeleteCommand(bot, dbConn, update)
	case "List":
		handleListVoices(bot, dbConn, chatID, userID)
	default:
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
	case "waiting_for_edit_selection":
		handleEditChoice(bot, dbConn, update)
	case "editing_voice":
		switch strings.ToLower(update.Message.Text) {
		case "name":
			SetUserState(update.Message.From.ID, "editing_voice_name")
			sendMessage(bot, update.Message.Chat.ID, "Enter a new name:")
		case "description":
			SetUserState(update.Message.From.ID, "editing_voice_description")
			sendMessage(bot, update.Message.Chat.ID, "Enter a new description:")
		default:
			sendMessage(bot, update.Message.Chat.ID, "Please select what you want to change (name/description).")
		}
	case "editing_voice_name":
		state := GetFSMState(userID)
		voiceID, _ := strconv.Atoi(state.Voice)
		db.UpdateVoiceField(dbConn, voiceID, "name", update.Message.Text)
		sendMessage(bot, update.Message.Chat.ID, "Name updated successfully!")
		SetUserState(update.Message.From.ID, "")
	case "editing_voice_description":
		state := GetFSMState(userID)
		voiceID, _ := strconv.Atoi(state.Voice)
		db.UpdateVoiceField(dbConn, voiceID, "description", update.Message.Text)
		sendMessage(bot, update.Message.Chat.ID, "Description updated successfully!")
		SetUserState(update.Message.From.ID, "")
	case "waiting_for_delete_selection":
		handleDeleteChoice(bot, dbConn, update)
	case "deleting_voice":
		state := GetFSMState(userID)
		voiceID, _ := strconv.Atoi(state.Voice)
		if update.Message.Text == "Yes" {
			db.DeleteVoice(dbConn, voiceID)
			sendMessage(bot, chatID, "Deletion successful")
		}
		SetUserState(userID, "")
		sendMessage(bot, chatID, "Cancelled")
	default:
		sendMessage(bot, chatID, "Unknown command or state.")
	}
}

func handleCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) {

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)

	switch update.Message.Command() {
	case "start", "open":
		msg.Text = "Keyboard is open"
		msg.ReplyMarkup = numericKeyboard
	case "close":
		msg.Text = "Keyboard is closed"
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	case "cancel":
		msg.Text = "Action cancelled"
		SetUserState(update.Message.From.ID, "")

	default:
		msg.Text = "I don't know that command"
		msg.ReplyMarkup = numericKeyboard
	}
	bot.Send(msg)
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

// The function `handleEditCommand` retrieves a user's recordings from a database and prompts the user
// to select a recording to edit.
func handleEditCommand(bot *tgbotapi.BotAPI, dbConn *sql.DB, update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID

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

// The function `handleEditChoice` processes user input for editing a voice recording in a chatbot
// application.
func handleEditChoice(bot *tgbotapi.BotAPI, dbConn *sql.DB, update tgbotapi.Update) {
	index, err := strconv.Atoi(update.Message.Text)
	if err != nil || index <= 0 {
		sendMessage(bot, update.Message.Chat.ID, "Please enter a valid number.")
		return
	}

	voices, err := db.GetUserVoices(dbConn, update.Message.From.ID)
	if err != nil || index > len(voices) {
		sendMessage(bot, update.Message.Chat.ID, "Invalid number. Try again.")
		return
	}

	selectedVoice := voices[index-1]
	SetUserState(update.Message.From.ID, "editing_voice")
	userStates[update.Message.From.ID].Voice = strconv.Itoa(selectedVoice.ID)

	sendMessage(bot, update.Message.Chat.ID, "You are editing the recording: "+selectedVoice.Name+". What would you like to change? (name/description)")
}

func handleDeleteCommand(bot *tgbotapi.BotAPI, dbConn *sql.DB, update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID

	voices, err := db.GetUserVoices(dbConn, userID)
	if err != nil {
		log.Printf("Error retrieving voices: %v", err)
		sendMessage(bot, chatID, "Error retrieving your recordings.")
		SetUserState(userID, "")
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

	response.WriteString("\nEnter the number of the recording you want to delete:")
	sendMessage(bot, chatID, response.String())
	SetUserState(userID, "waiting_for_delete_selection")
}

func handleDeleteChoice(bot *tgbotapi.BotAPI, dbConn *sql.DB, update tgbotapi.Update) {
	index, err := strconv.Atoi(update.Message.Text)
	if err != nil || index <= 0 {
		sendMessage(bot, update.Message.Chat.ID, "Please enter a valid number.")
		return
	}

	voices, err := db.GetUserVoices(dbConn, update.Message.From.ID)
	if err != nil || index > len(voices) {
		sendMessage(bot, update.Message.Chat.ID, "Invalid number. Try again.")
		return
	}

	selectedVoice := voices[index-1]
	SetUserState(update.Message.From.ID, "deleting_voice")
	userStates[update.Message.From.ID].Voice = strconv.Itoa(selectedVoice.ID)

	sendMessage(bot, update.Message.Chat.ID, "You are deleting the recording: "+selectedVoice.Name+". What would you like to continue? Yes/No")
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
