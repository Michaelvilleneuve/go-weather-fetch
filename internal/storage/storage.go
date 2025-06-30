package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
)

// Define structs for GeoJSON to reduce memory overhead
type Coordinate [2]float64

type Polygon struct {
	Type        string        `json:"type"`
	Coordinates [][][]float64 `json:"coordinates"`
}

type Properties struct {
	Value float64 `json:"value"`
}

type Feature struct {
	Type       string     `json:"type"`
	Geometry   Polygon    `json:"geometry"`
	Properties Properties `json:"properties"`
}

type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

func AnticipateExit() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		utils.Log("Cleaning up before exit...")
		CleanUpFiles("")
		os.Exit(0)
	}()
}


func CleanUpFiles(pattern string) {
	files, err := os.ReadDir("./tmp")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if len(pattern) > 0 && !strings.Contains(file.Name(), pattern) {
			continue
		}
		os.Remove("./tmp/" + file.Name())
	}
}


func Save(data [][]float64, processedFile ProcessedFile) (ProcessedFile, error) {
	const cellSizeKm = 1.1
	const cellHalfSizeDeg = (cellSizeKm / 2) / 111
	const batchSize = 1000 // Process data in batches to reduce memory usage

	utils.Log("Saving GeoJSON for " + processedFile.GetFileName())

	file, err := os.Create(processedFile.GetTmpGeoJSONFilePath())
	if err != nil {
		return processedFile, err
	}
	defer file.Close()

	// Start JSON manually to enable streaming
	file.WriteString(`{"type":"FeatureCollection","features":[`)

	totalBatches := (len(data) + batchSize - 1) / batchSize
	
	for batchIdx := 0; batchIdx < totalBatches; batchIdx++ {
		start := batchIdx * batchSize
		end := start + batchSize
		if end > len(data) {
			end = len(data)
		}

		batch := make([]Feature, 0, end-start)
		
		for i := start; i < end; i++ {
			point := data[i]
			if !geometry.IsPointInPolygon(geometry.Point{Lat: point[1], Lon: point[0]}) {
				continue
			}
			
			// Pre-calculate coordinates to reduce repeated calculations
			lonMin := point[0] - cellHalfSizeDeg
			lonMax := point[0] + cellHalfSizeDeg
			latMin := point[1] - cellHalfSizeDeg
			latMax := point[1] + cellHalfSizeDeg
			
			// Create polygon coordinates more efficiently
			coordinates := [][][]float64{{
				{lonMin, latMin},
				{lonMax, latMin},
				{lonMax, latMax},
				{lonMin, latMax},
				{lonMin, latMin},
			}}

			feature := Feature{
				Type: "Feature",
				Geometry: Polygon{
					Type:        "Polygon",
					Coordinates: coordinates,
				},
				Properties: Properties{
					Value: point[2],
				},
			}
			
			batch = append(batch, feature)
		}

		// Marshal and write batch
		batchJSON, err := json.Marshal(batch)
		if err != nil {
			return processedFile, err
		}

		// Remove the outer array brackets from batch JSON
		batchContent := string(batchJSON[1 : len(batchJSON)-1])
		
		if batchIdx > 0 && len(batchContent) > 0 {
			file.WriteString(",")
		}
		
		if len(batchContent) > 0 {
			file.WriteString(batchContent)
		}
		
		// Clear batch to free memory
		batch = nil
	}

	// Close JSON
	file.WriteString(`]}`)
	file.Sync()

	return convertToMBTiles(processedFile)
}

func convertToMBTiles(processedFile ProcessedFile) (ProcessedFile, error) {
	utils.Log("Tippecanoe command for " + processedFile.GetTmpGeoJSONFilePath())

	cmd := exec.Command("tippecanoe",
		"-o", processedFile.GetTmpMBTilesFilePath(),
		"--read-parallel",
		"--drop-densest-as-needed",
		"--force",
		"--minimum-zoom=5",
		"--maximum-zoom=9",
		processedFile.GetTmpGeoJSONFilePath(),
	)

	// Capture output to debug potential issues
	output, err := cmd.CombinedOutput()
	if err != nil {
		utils.Log(fmt.Sprintf("Tippecanoe error: %s\nOutput: %s", err.Error(), string(output)))
		return processedFile, fmt.Errorf("tippecanoe failed: %s", err)
	}

	utils.Log(fmt.Sprintf("Successfully generated %s", processedFile.GetTmpMBTilesFilePath()))

	// Remove the geojson file
	os.Remove(processedFile.GetTmpGeoJSONFilePath())

	return processedFile, nil
}

func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	err = dstFile.Sync()
	if err != nil {
		return err
	}

	return os.Remove(src)
}