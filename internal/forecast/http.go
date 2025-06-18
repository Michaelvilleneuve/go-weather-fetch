package forecast

import (
	"net/http"
)

func Serve() {
	for _, forecastPackage := range FORECAST_PACKAGES {
		for _, forecastGroup := range forecastPackage.Forecasts {
			http.HandleFunc("/" + forecastGroup.CommonName + ".json", func(w http.ResponseWriter, r *http.Request) {
				// Get the hour from the request
				hour := r.URL.Query().Get("hour")

				if hour == "" {
					http.Error(w, "Hour is required", http.StatusBadRequest)
					return
				}

				// Add a leading zero since filenames are like 00, 01, 02, etc.
				if len(hour) == 1 {
					hour = "0" + hour
				}

				w.Header().Set("Content-Type", "application/json"	)
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS, HEAD")

				w.WriteHeader(http.StatusOK)

				http.ServeFile(w, r, "storage/" + forecastGroup.CommonName + "_" + hour + ".json.gz")
			})
		}
	}	
}