package main

import (
	"os"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/forecast"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/server"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/storage"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
)


func main() {
	utils.LoadEnv()
	storage.AnticipateExit()
	
	if os.Getenv("WORKER") == "true" {
		// We still need to serve the /up endpoint even though we are a just aworker
		go server.Serve()
		forecast.StartFetching()
	} else {
		server.Serve()
	}
}