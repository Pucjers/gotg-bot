package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnvironment() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	if _, exists := os.LookupEnv("TELEGRAM_APITOKEN"); !exists {
		log.Fatal("Environment variable TELEGRAM_APITOKEN is required")
	}

	if _, exists := os.LookupEnv("CONNECTION_STRING"); !exists {
		log.Fatal("Environment variable CONNECTION_STRING is required")
	}
}
