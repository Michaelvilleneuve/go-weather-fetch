package utils

import (
	"log"
	"os"
	"fmt"
	"github.com/joho/godotenv"
)


func CleanUpFiles() {
	files, err := os.ReadDir("./tmp")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		os.Remove("./tmp/" + file.Name())
	}
}

func LoadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func Log(message string) {
	if os.Getenv("DEBUG") == "true" {
		fmt.Println(message)
	}
}