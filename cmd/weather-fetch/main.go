package main

import (
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/forecast"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/server"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/storage"
)


func main() {
	storage.AnticipateExit()
	go server.Serve()
	forecast.StartFetching()
}