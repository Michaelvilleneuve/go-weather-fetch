package storage

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
)

const (
	MinLon = -12.736815506536061
	MaxLon = 12.998159721174034
	MinLat = 37.33132975722407
	MaxLat = 55.94846979868936
	
	// Pre-caching configuration
	MinZoom = 5
	MaxZoom = 6
	BatchSize = 100
)

// TileCoordinate represents a tile coordinate
type TileCoordinate struct {
	Z, X, Y int
}

// PreCacheTiles performs pre-caching of tiles for the given processed file
func PreCacheTiles(processedFile ProcessedFile) {
	utils.Log("Pre-caching tiles for " + processedFile.GetFileName())
	
	// Generate all tile coordinates within the bounding box
	tiles := generateTileCoordinates(MinZoom, MaxZoom, MinLon, MaxLon, MinLat, MaxLat)
	
	utils.Log(fmt.Sprintf("Total tiles to pre-cache: %d", len(tiles)))
	utils.Log(fmt.Sprintf("First few tiles: %+v", tiles[:5]))
	utils.Log(fmt.Sprintf("Bounding box: MinLon=%f, MaxLon=%f, MinLat=%f, MaxLat=%f", MinLon, MaxLon, MinLat, MaxLat))
	
	// Get the base URL for tile requests
	baseURL := getBaseURL()
	
	// Pre-cache tiles in batches
	preCacheTilesInBatches(tiles, processedFile, baseURL, BatchSize)
}

// generateTileCoordinates generates all tile coordinates within the bounding box for the specified zoom levels
func generateTileCoordinates(minZoom, maxZoom int, minLon, maxLon, minLat, maxLat float64) []TileCoordinate {
	var tiles []TileCoordinate
	
	for zoom := minZoom; zoom <= maxZoom; zoom++ {
		// Convert lat/lon bounds to tile coordinates
		minTileX, maxTileY := degToTileCoord(minLon, maxLat, zoom)
		maxTileX, minTileY := degToTileCoord(maxLon, minLat, zoom)
		
		utils.Log(fmt.Sprintf("Zoom %d: X range [%d-%d], Y range [%d-%d]", zoom, minTileX, maxTileX, minTileY, maxTileY))
		
		// Generate all tiles in the bounding box
		for x := minTileX; x <= maxTileX; x++ {
			for y := minTileY; y <= maxTileY; y++ {
				tiles = append(tiles, TileCoordinate{Z: zoom, X: x, Y: y})
			}
		}
	}
	
	return tiles
}

// degToTileCoord converts lat/lon coordinates to tile coordinates for a given zoom level
func degToTileCoord(lon, lat float64, zoom int) (int, int) {
	// Convert longitude to tile X coordinate
	tileX := int(math.Floor((lon + 180.0) / 360.0 * math.Pow(2.0, float64(zoom))))
	
	// Convert latitude to tile Y coordinate
	latRad := lat * math.Pi / 180.0
	n := math.Pow(2.0, float64(zoom))
	tileY := int(math.Floor((1.0 - math.Asinh(math.Tan(latRad))/math.Pi) / 2.0 * n))
	
	// Flip Y coordinate to match web mercator system
	tileY = int(n) - 1 - tileY
	
	return tileX, tileY
}

// preCacheTilesInBatches pre-caches tiles in batches to avoid overwhelming the server
func preCacheTilesInBatches(tiles []TileCoordinate, processedFile ProcessedFile, baseURL string, batchSize int) {
	totalTiles := len(tiles)
	processedCount := 0
	
	for i := 0; i < totalTiles; i += batchSize {
		end := i + batchSize
		if end > totalTiles {
			end = totalTiles
		}
		
		batch := tiles[i:end]
		
		// Process batch in parallel
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, batchSize)
		
		for _, tile := range batch {
			wg.Add(1)
			semaphore <- struct{}{}
			
			go func(t TileCoordinate) {
				defer wg.Done()
				defer func() { <-semaphore }()
				
				url := buildTileURL(baseURL, processedFile, t)
				fetchTile(url)
			}(tile)
		}
		
		wg.Wait()
		
		processedCount += len(batch)
		utils.Log(fmt.Sprintf("Pre-cached batch: %d/%d tiles", processedCount, totalTiles))
		
		// Small delay between batches to prevent overwhelming the server
		time.Sleep(100 * time.Millisecond)
	}
	
	utils.Log(fmt.Sprintf("Pre-caching completed: %d tiles processed", processedCount))
}

// buildTileURL constructs the appropriate tile URL based on the processed file format
func buildTileURL(baseURL string, processedFile ProcessedFile, tile TileCoordinate) string {
	hourStr := fmt.Sprintf("%02d", parseHour(processedFile.Hour))
	
	if processedFile.Format == "cog" {
		return fmt.Sprintf("%s/tiles/cog/%s/%s/%d/%d/%d.png", 
			baseURL, processedFile.Layer, hourStr, tile.Z, tile.X, tile.Y)
	}
	
	// Default to MBTiles format
	return fmt.Sprintf("%s/tiles/arome/%s/%s/%d/%d/%d.pbf", 
		baseURL, processedFile.Layer, hourStr, tile.Z, tile.X, tile.Y)
}

// parseHour converts hour string to integer, handling potential parsing errors
func parseHour(hourStr string) int {
	hour, err := strconv.Atoi(hourStr)
	if err != nil {
		utils.Log("Error parsing hour: " + err.Error())
		return 1 // Default to hour 1
	}
	return hour
}

func fetchTile(url string) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	resp, err := client.Get(url)
	if err != nil {
		utils.Log("Error fetching tile " + url + ": " + err.Error())
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		utils.Log(fmt.Sprintf("Non-200 status for tile %s: %d", url, resp.StatusCode))
		return
	}
}

func getBaseURL() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	
	return fmt.Sprintf("http://localhost:%s", port)
} 