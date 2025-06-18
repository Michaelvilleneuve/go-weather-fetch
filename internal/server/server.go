package server

import (
	"net/http"
	"os"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/forecast"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
)

func Serve() {
	forecast.Serve()
	http.HandleFunc("/up", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ha ha ha ha staying alive"))
	})

	utils.Log("Serving forecast on port " + os.Getenv("PORT"))
	http.ListenAndServe(":" + os.Getenv("PORT"), nil)
}