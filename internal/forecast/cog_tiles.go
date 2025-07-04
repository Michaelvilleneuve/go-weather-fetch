package forecast

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/arome"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
)

// TileCache holds cached PNG tiles
type TileCache struct {
	mu    sync.RWMutex
	tiles map[string]CachedTile
}

type CachedTile struct {
	data      []byte
	timestamp time.Time
	fileHash  string
}

var (
	tileCache = &TileCache{
		tiles: make(map[string]CachedTile),
	}
	// Cache tiles for 60 minutes
	tileCacheDuration = 60 * time.Minute
)

func serveCOGTiles() {
	// Start cache cleanup routine
	go cleanupTileCache()
	
	for _, aromeLayer := range arome.Configuration().GetLayers() {
		for hour := 1; hour <= 51; hour++ {
			commonName := aromeLayer.CommonName
			hourStr := fmt.Sprintf("%02d", hour)
			
			// Create route: /tiles/cog/{layer}/{hour}/{z}/{x}/{y}.png
			pattern := fmt.Sprintf("/tiles/cog/%s/%s/", commonName, hourStr)
			http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
				setCORSHeaders(w, r)
				handleCOGTileRequest(w, r, commonName, hourStr)
			})
		}
	}
}

func cleanupTileCache() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		tileCache.mu.Lock()
		now := time.Now()
		for key, tile := range tileCache.tiles {
			if now.Sub(tile.timestamp) > tileCacheDuration {
				delete(tileCache.tiles, key)
			}
		}
		tileCache.mu.Unlock()
	}
}

func handleCOGTileRequest(w http.ResponseWriter, r *http.Request, layer, hour string) {
	setCORSHeaders(w, r)
	// Parse URL: /tiles/cog/{layer}/{hour}/{z}/{x}/{y}.png
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 7 {
		http.Error(w, "Invalid tile path", http.StatusBadRequest)
		return
	}

	layer = parts[3]
	
	z, err := strconv.Atoi(parts[5])
	if err != nil {
		http.Error(w, "Invalid zoom level", http.StatusBadRequest)
		return
	}
	
	x, err := strconv.Atoi(parts[6])
	if err != nil {
		http.Error(w, "Invalid x coordinate", http.StatusBadRequest)
		return
	}
	
	yParts := strings.Split(parts[7], ".")
	if len(yParts) < 2 || (yParts[1] != "png" && yParts[1] != "json") {
		http.Error(w, "Invalid y coordinate or file extension", http.StatusBadRequest)
		return
	}
	
	y, err := strconv.Atoi(yParts[0])
	if err != nil {
		http.Error(w, "Invalid y coordinate", http.StatusBadRequest)
		return
	}
	
	// Find the most recent COG file for this layer and hour
	cogFile := findMostRecentCOGFile(layer, hour)
	if cogFile == "" {
		http.Error(w, "COG file not found", http.StatusNotFound)
		return
	}

	if yParts[1] == "json" {
		jsonData, err := extractJSONDataFromCOG(cogFile, layer, z, x, y)
		if err != nil {
			http.Error(w, "Failed to generate JSON data" + err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("X-Cache", "HIT")
		w.Write(jsonData)
		return
	}
	
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%d:%d:%d", cogFile, z, x, y)
	if cachedTile := getCachedTile(cacheKey, cogFile); cachedTile != nil {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("X-Cache", "HIT")
		w.Write(cachedTile)
		return
	}
	
	// Generate PNG tile from COG using optimized gdalwarp
	pngData, err := extractPNGTileFromCOG(cogFile, z, x, y)
	if err != nil {
		utils.Log("Error extracting PNG tile: " + err.Error())
		http.Error(w, "Failed to generate tile", http.StatusInternalServerError)
		return
	}
	
	// Cache the tile
	setCachedTile(cacheKey, pngData, cogFile)
	
	// Serve PNG tile
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	w.Header().Set("X-Cache", "MISS")
	w.Write(pngData)
}

func getCachedTile(cacheKey, cogFile string) []byte {
	tileCache.mu.RLock()
	defer tileCache.mu.RUnlock()
	
	if tile, exists := tileCache.tiles[cacheKey]; exists {
		// Check if the COG file has been modified since caching
		if fileInfo, err := os.Stat(cogFile); err == nil {
			currentHash := getFileHash(cogFile, fileInfo.ModTime())
			if tile.fileHash == currentHash && time.Since(tile.timestamp) < tileCacheDuration {
				return tile.data
			}
		}
	}
	return nil
}

func setCachedTile(cacheKey string, data []byte, cogFile string) {
	if fileInfo, err := os.Stat(cogFile); err == nil {
		tileCache.mu.Lock()
		defer tileCache.mu.Unlock()
		
		tileCache.tiles[cacheKey] = CachedTile{
			data:      data,
			timestamp: time.Now(),
			fileHash:  getFileHash(cogFile, fileInfo.ModTime()),
		}
	}
}

func getFileHash(filepath string, modTime time.Time) string {
	hasher := md5.New()
	hasher.Write([]byte(filepath))
	hasher.Write([]byte(modTime.String()))
	return hex.EncodeToString(hasher.Sum(nil))
}

func findMostRecentCOGFile(layer, hour string) string {
	// Pattern: arome_{run}_{layer}_{hour}_cog.tif
	pattern := fmt.Sprintf("storage/arome_*_%s_%s_cog.tif", layer, hour)
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		return ""
	}
	
	// Sort files by modification time, return most recent
	var mostRecent string
	var mostRecentTime time.Time
	
	for _, file := range files {
		if info, err := os.Stat(file); err == nil {
			if mostRecent == "" || info.ModTime().After(mostRecentTime) {
				mostRecent = file
				mostRecentTime = info.ModTime()
			}
		}
	}
	
	return mostRecent
}

func extractPNGTileFromCOG(cogFile string, z, x, y int) ([]byte, error) {
	// Calculate Web Mercator bounds for the tile
	bounds := tileToWebMercatorBounds(z, x, y)
	
	// Use gdal_translate to extract RGBA bands and create PNG tile in one command
	cmd := exec.Command("gdal_translate",
		"-of", "PNG",
		"-b", "1", "-b", "2", "-b", "3", "-b", "4", // Only RGBA bands, skip band 5 (value)
		"-projwin", 
		fmt.Sprintf("%.10f", bounds.minX), // ulx (upper left x)
		fmt.Sprintf("%.10f", bounds.maxY), // uly (upper left y) 
		fmt.Sprintf("%.10f", bounds.maxX), // lrx (lower right x)
		fmt.Sprintf("%.10f", bounds.minY), // lry (lower right y)
		"-projwin_srs", "EPSG:3857", // Projection window coordinates are in Web Mercator
		"-outsize", "256", "256",    // Output size 256x256
		"-r", "cubic",               // Cubic resampling
		"-co", "WORLDFILE=NO",       // Don't create world file
		"-q",                        // Quiet mode
		cogFile,
		"/vsistdout/",               // Output to stdout
	)

	// Set environment variables for better performance
	cmd.Env = append(os.Environ(),
		"GDAL_DISABLE_READDIR_ON_OPEN=EMPTY_DIR",
		"GDAL_TIFF_INTERNAL_MASK=YES",
	)
	
	pngData, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PNG tile: %s", err)
	}
	
	return pngData, nil
}

// COGInfo holds information about a COG file
type COGInfo struct {
	Width, Height int
	PixelSizeX, PixelSizeY float64
	MinX, MinY, MaxX, MaxY float64
}

// GDALInfo structures for parsing gdalinfo JSON output
type GDALInfo struct {
	Size         []int               `json:"size"`
	GeoTransform []float64           `json:"geoTransform"`
	Bands        []map[string]interface{} `json:"bands"`
	CornerCoordinates map[string][]float64 `json:"cornerCoordinates"`
}

type TileBounds struct {
	minX, minY, maxX, maxY float64
}

func tileToWebMercatorBounds(z, x, y int) TileBounds {
	// Web Mercator constants
	const earthRadius = 6378137.0
	const originShift = math.Pi * earthRadius
	
	// Convert tile coordinates to Web Mercator bounds
	tilePower := 1 << uint(z)
	res := (2 * originShift) / float64(tilePower) / 256.0
	
	minX := float64(x*256)*res - originShift
	maxY := originShift - float64(y*256)*res
	maxX := float64((x+1)*256)*res - originShift  
	minY := originShift - float64((y+1)*256)*res
	
	return TileBounds{
		minX: minX,
		minY: minY, 
		maxX: maxX,
		maxY: maxY,
	}
}

func extractJSONDataFromCOG(cogFile string, layer string, z, x, y int) ([]byte, error) {
	// Calculate Web Mercator bounds for the tile
	bounds := tileToWebMercatorBounds(z, x, y)

	// Calculate center coordinates of the tile
	centerX := (bounds.minX + bounds.maxX) / 2
	centerY := (bounds.minY + bounds.maxY) / 2

	// Use gdallocationinfo WITHOUT -geoloc flag, using projected coordinates
	cmd := exec.Command("gdallocationinfo", 
		"-valonly",
		"-b", "5", // Read from band 5 (value band)
		"-l_srs", "EPSG:3857", // Specify that input coordinates are in Web Mercator
		cogFile,
		fmt.Sprintf("%.10f", centerX),
		fmt.Sprintf("%.10f", centerY),
	)

	// Set environment variables for better performance
	cmd.Env = append(os.Environ(),
		"GDAL_DISABLE_READDIR_ON_OPEN=EMPTY_DIR",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get value from COG: %s", err)
	}

	// Parse the value from output
	valueStr := strings.TrimSpace(string(output))
	weatherValue, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse value: %s", err)
	}

	// Create JSON response
	response := map[string]interface{}{
		"value": weatherValue,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %s", err)
	}

	return jsonData, nil
}