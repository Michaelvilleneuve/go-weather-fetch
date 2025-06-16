package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"github.com/joho/godotenv"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/forecast"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/grib"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
)


func main() {
	loadEnv()
	availableDatetimes := forecast.GetAvailableRunDates()

	for _, dt := range availableDatetimes {

		if !forecast.CheckIfAllForecastsHoursAreAvailable(dt) {
			fmt.Printf("No forecast yet for %s\n", dt)
			continue
		}

		err := os.WriteFile("last_download.txt", []byte(dt), 0644)

		if err != nil {
			fmt.Printf("Error writing last_download.txt: %v\n", err)
		}

		var (
			wg sync.WaitGroup
		)

		for _, hour := range forecast.GetAvailableHours(dt) {
			wg.Add(1)
			go func(h string) {
				defer wg.Done()
				processSingleForecast(dt, h)
			}(hour)
		}

		wg.Wait()

		cleanUpFiles()

		os.Exit(0)
	}
}

func processSingleForecast(dt string, hour string) (string, error) {
	filename, err := forecast.GetSingleForecast(dt, hour)
	if err != nil {
		return "", err
	}

	fmt.Println("Extracting GRIB data...")
	allPoints, err := grib.ExtractGribData(filename)
	if err != nil {
		log.Fatal("Error extracting GRIB data:", err)
	}

	fmt.Printf("Total points extracted: %d\n", len(allPoints))

	polygon := []geometry.Point{
		{Lat: 39.7153328, Lon: 1.1861908},
		{Lat: 39.7097536, Lon: 0.3860986},
		{Lat: 39.7049828, Lon: -1.2260914},
		{Lat: 37.8525431, Lon: -1.2438369},
		{Lat: 37.8358186, Lon: 1.1625552},
	}

	fmt.Println("Filtering points within polygon...")
	pointsInPolygon := geometry.FilterPointsByPolygon(allPoints, polygon)
	fmt.Printf("Points within polygon: %d\n", len(pointsInPolygon))

	// fmt.Println("\nSample of filtered points:")
	// for i, point := range pointsInPolygon {
	// 	fmt.Printf("Point %d: Lat=%.6f, Lon=%.6f, Value=%.6f\n",
	// 		i+1, point.Lat, point.Lon, point.Value)
	// }

	// Array of array
	allData := [][]float64{}

	if len(pointsInPolygon) > 0 {
		for _, point := range pointsInPolygon {
			value := 0.0
			if point.Value < 9999 {
				value = point.Value
			}

			allData = append(allData, []float64{point.Lon, point.Lat, value})
		}
	}

	api.SendToApi(allData, hour, dt)

	return "", nil
}

func cleanUpFiles() {
	files, err := os.ReadDir("./tmp")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		os.Remove(file.Name())
	}
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}