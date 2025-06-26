package forecast

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
	_ "github.com/mattn/go-sqlite3"
)

func Serve() {
	for _, forecastPackage := range FORECAST_PACKAGES {
		for _, forecastGroup := range forecastPackage.Forecasts {
			for hour := 1; hour <= 51; hour++ {
				// IMPORTANT: Capture des variables pour éviter les problèmes de closure
				commonName := forecastGroup.CommonName
				hourStr := fmt.Sprintf("%02d", hour) // Format avec zéro devant : 01, 02, etc.
				
				http.HandleFunc("/tiles/" + commonName + "/" + hourStr + "/", func(w http.ResponseWriter, r *http.Request) {
					// Add CORS headers
					w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

					// Handle preflight requests
					if r.Method == "OPTIONS" {
						w.WriteHeader(http.StatusOK)
						return
					}

					filePath := "storage/" + commonName + "_" + hourStr + ".geojson.mbtiles"
					
					// Vérifier que le fichier existe et n'est pas vide
					if fileInfo, err := os.Stat(filePath); err != nil {
						utils.Log("File not found: " + filePath + " - " + err.Error())
						http.Error(w, "MBTiles file not found", http.StatusNotFound)
						return
					} else if fileInfo.Size() == 0 {
						utils.Log("Empty file: " + filePath)
						http.Error(w, "MBTiles file is empty", http.StatusInternalServerError)
						return
					}
					
					db, err := sql.Open("sqlite3", filePath)
					if err != nil {
						utils.Log("Error opening database: " + err.Error())
						http.Error(w, "Failed to open MBTiles", http.StatusInternalServerError)
						return
					}
					defer db.Close()

					// Test de connexion et vérification des tables
					var tableCount int
					err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='map'").Scan(&tableCount)
					if err != nil || tableCount == 0 {
						utils.Log("Table 'map' not found in " + filePath)
						http.Error(w, "Invalid MBTiles format", http.StatusInternalServerError)
						return
					}

					tileHandler(w, r, db)
				})
			}
		}
	}	
}

func tileHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// URL: /tiles/{forecast_group}/{hour}/{z}/{x}/{y}.pbf
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid tile path", http.StatusBadRequest)
		return
	}
	
	z, _ := strconv.Atoi(parts[4])
	x, _ := strconv.Atoi(parts[5])
	yParts := strings.Split(parts[6], ".")
	y, _ := strconv.Atoi(yParts[0])
	format := yParts[1]
	
	utils.Log("format: " + format)
	utils.Log("z: " + strconv.Itoa(z))
	utils.Log("x: " + strconv.Itoa(x))
	utils.Log("y: " + strconv.Itoa(y))
	
	// Flip Y for TMS
	y = (1 << uint(z)) - 1 - y

	var tileData []byte
	err := db.QueryRow(`
		SELECT images.tile_data 
		FROM map 
		JOIN images ON map.tile_id = images.tile_id 
		WHERE map.zoom_level = ? AND map.tile_column = ? AND map.tile_row = ?
	`, z, x, y).Scan(&tileData)

	if err != nil {
		utils.Log("Error: " + err.Error())
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Header().Set("Content-Encoding", "gzip")

	w.Write(tileData)
}