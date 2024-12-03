package file

import (
	"fmt"
	"io"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// The function `DownloadVoiceFile` downloads a voice file from Telegram using a bot API and saves it
// to a specified destination folder.
func DownloadVoiceFile(bot *tgbotapi.BotAPI, fileID string) (string, error) {
	file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", fmt.Errorf("error getting file: %v", err)
	}

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", bot.Token, file.FilePath)
	destinationFolder := "voices/"
	if err := os.MkdirAll(destinationFolder, os.ModePerm); err != nil {
		return "", fmt.Errorf("error creating directory: %v", err)
	}

	filePath := destinationFolder + file.FileID + ".ogg"
	outFile, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("error creating file: %v", err)
	}
	defer outFile.Close()

	resp, err := http.Get(fileURL)
	if err != nil {
		return "", fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("error saving file: %v", err)
	}

	return filePath, nil
}
