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

// The main function loads environment configuration, connects to a database, and runs a bot using the
// database connection.
func main() {
	config.LoadEnvironment()

	dbConn, err := db.ConnectDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer dbConn.Close()

	bot.Run(dbConn)
}
