package forecast

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/arome"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
	_ "github.com/mattn/go-sqlite3"
)

func Serve() {
	serveTilesAsPbf()
	http.HandleFunc("/metadata.json", metadataHandler)
}

func metadataHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	// Read the current run datetime from storage
	content, err := os.ReadFile("storage/SP1_current_run_datetime.txt")
	if err != nil {
		utils.Log("Error reading current run datetime: " + err.Error())
		w.Write([]byte(`{"run_hour": ""}`))
		return
	}

	// Trim any whitespace and write the datetime
	datetime := strings.TrimSpace(string(content))
	parsedDatetime, err := time.Parse("2006-01-02T15:04:05Z", datetime)
	if err != nil {
		utils.Log("Error parsing datetime: " + err.Error())
		w.Write([]byte(`{"run_hour": ""}`))
		return
	}
	datetimeStartHour := parsedDatetime.Add(time.Hour * 1).Format("2006-01-02T15:04:05Z")
	w.Write([]byte(fmt.Sprintf(`{"run_hour": "%s", "start_hour": "%s"}`, datetime, datetimeStartHour)))
}

func tileHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
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

func setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := os.Getenv("HOST_ORIGIN")
	if origin == "" {
		origin = "http://localhost:3000"
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
}

func serveTilesAsPbf() {
	for _, aromeLayer := range arome.Configuration().GetLayers() {
		for hour := 1; hour <= 51; hour++ {
			commonName := aromeLayer.CommonName
			hourStr := fmt.Sprintf("%02d", hour) // Format avec zéro devant : 01, 02, etc.
			
			http.HandleFunc("/tiles/" + commonName + "/" + hourStr + "/", func(w http.ResponseWriter, r *http.Request) {
				// Add CORS headers
				setCORSHeaders(w, r)

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