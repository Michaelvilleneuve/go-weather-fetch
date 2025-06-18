package utils

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning: No .env file found, using environment variables")
	}
}

func Log(message string) {
	if os.Getenv("DEBUG") == "true" {
		fmt.Println(message)
	}
}