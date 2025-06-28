package server

import (
	"net/http"
	"os"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/forecast"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
)

func Serve() {
	utils.LoadEnv()
	if os.Getenv("WORKER") != "false" || os.Getenv("SERVER") == "true" {
		forecast.Serve()
	}
	
	http.HandleFunc("/up", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ha ha ha ha staying alive"))
	})

	http.Handle("/", http.FileServer(http.Dir("public")))

	utils.Log("Serving forecast on port " + os.Getenv("PORT"))
	http.ListenAndServe(":" + os.Getenv("PORT"), nil)
}