package main

import (
	"log"

	"gotg-bot/config"
	"gotg-bot/db"
	"gotg-bot/internal/bot"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {
	config.LoadEnvironment()

	dbConn, err := db.ConnectDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer dbConn.Close()

	bot.Run(dbConn)
}
