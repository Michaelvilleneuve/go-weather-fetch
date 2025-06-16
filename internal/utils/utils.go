package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)


func CleanUpFiles() {
	files, err := os.ReadDir("./tmp")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		os.Remove(file.Name())
	}
}

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}