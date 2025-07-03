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
	// Cache tiles for 30 minutes
	tileCacheDuration = 30 * time.Minute
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
	if len(yParts) < 2 || yParts[1] != "png" {
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
	
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%d:%d:%d", cogFile, z, x, y)
	if cachedTile := getCachedTile(cacheKey, cogFile); cachedTile != nil {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("X-Cache", "HIT")
		w.Write(cachedTile)
		return
	}
	
	// Generate PNG tile from COG
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
	
	// Get COG file resolution information
	cogInfo, err := getCOGInfo(cogFile)
	if err != nil {
		utils.Log("Warning: Could not get COG info, using default tile size: " + err.Error())
		// Fallback to original behavior
		return extractPNGTileWithFixedSizeOptimized(cogFile, bounds)
	}
	
	// Calculate appropriate output size based on zoom level and native resolution
	outputWidth, outputHeight := calculateOptimalTileSize(bounds, cogInfo, z)
	
	// Choose resampling method based on zoom level
	resamplingMethod := "cubic"
	if z > 10 {
		// At high zoom levels, use nearest neighbor to preserve individual pixels
		resamplingMethod = "near"
	}
	
	// Use optimized gdalwarp with memory output
	return extractPNGTileInMemory(cogFile, bounds, outputWidth, outputHeight, resamplingMethod)
}

func extractPNGTileInMemory(cogFile string, bounds TileBounds, outputWidth, outputHeight int, resamplingMethod string) ([]byte, error) {
	// Use gdalwarp with direct stdout output for better performance
	cmd := exec.Command("gdalwarp",
		"-of", "PNG",
		"-dstalpha", // Add alpha band to output
		"-te", 
		fmt.Sprintf("%.10f", bounds.minX), // xmin
		fmt.Sprintf("%.10f", bounds.minY), // ymin
		fmt.Sprintf("%.10f", bounds.maxX), // xmax
		fmt.Sprintf("%.10f", bounds.maxY), // ymax
		"-te_srs", "EPSG:3857", // Target extent coordinates are in Web Mercator
		"-t_srs", "EPSG:3857",  // Output in Web Mercator
		"-ts", fmt.Sprintf("%d", outputWidth), fmt.Sprintf("%d", outputHeight), // Dynamic tile size
		"-r", resamplingMethod, // Dynamic resampling method
		"-co", "WORLDFILE=NO", // Don't create world file
		"-dstnodata", "0", // Set transparent pixels to 0
		"-multi", // Use multiple threads
		"-wo", "NUM_THREADS=ALL_CPUS", // Use all available CPUs
		"-ovr", "NONE", // Don't use overviews for small tiles
		cogFile,
		"/vsistdout/", // Output directly to stdout
	)
	
	pngData, err := cmd.Output()
	if err != nil {
		// Fallback to temporary file approach if stdout fails
		return extractPNGTileWithTempFile(cogFile, bounds, outputWidth, outputHeight, resamplingMethod)
	}
	
	return pngData, nil
}

func extractPNGTileWithTempFile(cogFile string, bounds TileBounds, outputWidth, outputHeight int, resamplingMethod string) ([]byte, error) {
	// Create a temporary file with better naming
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("tile_%d_%d_%d_%d.png", time.Now().UnixNano(), outputWidth, outputHeight, os.Getpid()))
	defer func() {
		// Ensure cleanup even if there's an error
		os.Remove(tmpFile)
	}()
	
	// Use gdalwarp with optimized parameters
	cmd := exec.Command("gdalwarp",
		"-of", "PNG",
		"-dstalpha", // Add alpha band to output
		"-te", 
		fmt.Sprintf("%.10f", bounds.minX), // xmin
		fmt.Sprintf("%.10f", bounds.minY), // ymin
		fmt.Sprintf("%.10f", bounds.maxX), // xmax
		fmt.Sprintf("%.10f", bounds.maxY), // ymax
		"-te_srs", "EPSG:3857", // Target extent coordinates are in Web Mercator
		"-t_srs", "EPSG:3857",  // Output in Web Mercator
		"-ts", fmt.Sprintf("%d", outputWidth), fmt.Sprintf("%d", outputHeight), // Dynamic tile size
		"-r", resamplingMethod, // Dynamic resampling method
		"-co", "WORLDFILE=NO", // Don't create world file
		"-dstnodata", "0", // Set transparent pixels to 0
		"-multi", // Use multiple threads
		"-wo", "NUM_THREADS=ALL_CPUS", // Use all available CPUs
		"-ovr", "NONE", // Don't use overviews for small tiles
		"-q", // Quiet mode for better performance
		cogFile,
		tmpFile,
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gdalwarp failed: %s\nOutput: %s", err, string(output))
	}
	
	// Read the generated PNG file
	pngData, err := os.ReadFile(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read generated PNG: %s", err)
	}
	
	return pngData, nil
}

func extractPNGTileWithFixedSizeOptimized(cogFile string, bounds TileBounds) ([]byte, error) {
	// Optimized fallback function with direct stdout
	cmd := exec.Command("gdalwarp",
		"-of", "PNG",
		"-dstalpha", // Add alpha band to output
		"-te", 
		fmt.Sprintf("%.10f", bounds.minX),
		fmt.Sprintf("%.10f", bounds.minY),
		fmt.Sprintf("%.10f", bounds.maxX),
		fmt.Sprintf("%.10f", bounds.maxY),
		"-te_srs", "EPSG:3857",
		"-t_srs", "EPSG:3857",
		"-ts", "256", "256",
		"-r", "cubic",
		"-co", "WORLDFILE=NO", // Don't create world file
		"-dstnodata", "0", // Set transparent pixels to 0
		"-multi", // Use multiple threads
		"-wo", "NUM_THREADS=ALL_CPUS", // Use all available CPUs
		"-q", // Quiet mode for better performance
		cogFile,
		"/vsistdout/", // Output directly to stdout
	)
	
	pngData, err := cmd.Output()
	if err != nil {
		// Fallback to temporary file if stdout fails
		return extractPNGTileWithTempFileFallback(cogFile, bounds)
	}
	
	return pngData, nil
}

func extractPNGTileWithTempFileFallback(cogFile string, bounds TileBounds) ([]byte, error) {
	// Final fallback with temporary file
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("tile_fallback_%d_%d.png", time.Now().UnixNano(), os.Getpid()))
	defer os.Remove(tmpFile)
	
	cmd := exec.Command("gdalwarp",
		"-of", "PNG",
		"-dstalpha", // Add alpha band to output
		"-te", 
		fmt.Sprintf("%.10f", bounds.minX),
		fmt.Sprintf("%.10f", bounds.minY),
		fmt.Sprintf("%.10f", bounds.maxX),
		fmt.Sprintf("%.10f", bounds.maxY),
		"-te_srs", "EPSG:3857",
		"-t_srs", "EPSG:3857",
		"-ts", "256", "256",
		"-r", "cubic",
		"-co", "WORLDFILE=NO", // Don't create world file
		"-dstnodata", "0", // Set transparent pixels to 0
		"-multi", // Use multiple threads
		"-wo", "NUM_THREADS=ALL_CPUS", // Use all available CPUs
		"-q", // Quiet mode
		cogFile,
		tmpFile,
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gdalwarp failed: %s\nOutput: %s", err, string(output))
	}
	
	pngData, err := os.ReadFile(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read PNG from temp file: %s", err)
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

func getCOGInfo(cogFile string) (*COGInfo, error) {
	// Use gdalinfo to get COG file information
	cmd := exec.Command("gdalinfo", "-json", cogFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gdalinfo failed: %s", err)
	}
	
	// Parse the JSON output
	var gdalInfo GDALInfo
	err = json.Unmarshal(output, &gdalInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gdalinfo JSON: %s", err)
	}
	
	// Extract basic information
	if len(gdalInfo.Size) != 2 {
		return nil, fmt.Errorf("invalid size information in COG file")
	}
	
	width := gdalInfo.Size[0]
	height := gdalInfo.Size[1]
	
	// Extract pixel size from geotransform
	// GeoTransform format: [originX, pixelWidth, 0, originY, 0, pixelHeight]
	var pixelSizeX, pixelSizeY float64 = 0.01, 0.01 // defaults
	if len(gdalInfo.GeoTransform) == 6 {
		pixelSizeX = math.Abs(gdalInfo.GeoTransform[1])
		pixelSizeY = math.Abs(gdalInfo.GeoTransform[5])
	}
	
	// Extract corner coordinates
	var minX, minY, maxX, maxY float64
	if upperLeft, ok := gdalInfo.CornerCoordinates["upperLeft"]; ok && len(upperLeft) == 2 {
		minX = upperLeft[0]
		maxY = upperLeft[1]
	}
	if lowerRight, ok := gdalInfo.CornerCoordinates["lowerRight"]; ok && len(lowerRight) == 2 {
		maxX = lowerRight[0]
		minY = lowerRight[1]
	}
	
	return &COGInfo{
		Width: width,
		Height: height,
		PixelSizeX: pixelSizeX,
		PixelSizeY: pixelSizeY,
		MinX: minX,
		MinY: minY,
		MaxX: maxX,
		MaxY: maxY,
	}, nil
}

func calculateOptimalTileSize(bounds TileBounds, cogInfo *COGInfo, zoom int) (int, int) {
	// For Web Mercator to WGS84 coordinate conversion
	// Web Mercator bounds are in meters, need to convert to degrees
	
	// Convert Web Mercator bounds to latitude/longitude
	minLon := bounds.minX * 180.0 / (math.Pi * 6378137.0)
	maxLon := bounds.maxX * 180.0 / (math.Pi * 6378137.0)
	
	// For latitude, the conversion is more complex due to Mercator projection
	minLat := math.Atan(math.Sinh(bounds.minY / 6378137.0)) * 180.0 / math.Pi
	maxLat := math.Atan(math.Sinh(bounds.maxY / 6378137.0)) * 180.0 / math.Pi
	
	// Calculate the geographic extent of the tile
	tileWidthDegrees := maxLon - minLon
	tileHeightDegrees := maxLat - minLat
	
	// Calculate how many native pixels would fit in this tile
	nativePixelsX := int(math.Abs(tileWidthDegrees / cogInfo.PixelSizeX))
	nativePixelsY := int(math.Abs(tileHeightDegrees / cogInfo.PixelSizeY))
	
	// At high zoom levels (>10), try to preserve native resolution
	if zoom > 10 {
		// Ensure we don't create tiles that are too large (max 1024x1024)
		maxSize := 1024
		if nativePixelsX > maxSize {
			nativePixelsX = maxSize
		}
		if nativePixelsY > maxSize {
			nativePixelsY = maxSize
		}
		
		// Ensure minimum size (at least 64x64)
		if nativePixelsX < 64 {
			nativePixelsX = 64
		}
		if nativePixelsY < 64 {
			nativePixelsY = 64
		}
		
		return nativePixelsX, nativePixelsY
	}
	
	// For lower zoom levels, use standard 256x256
	return 256, 256
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