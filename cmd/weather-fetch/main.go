package main

import (
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/forecast"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/server"
)


func main() {
	utils.LoadEnv()
	go server.Serve()
	forecast.StartFetching()
}