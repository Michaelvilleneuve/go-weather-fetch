package main

import (
	"fmt"
	"log"
	"os"
	"sync"
)


func main() {
	availableDatetimes := getAvailableDatetimes()

	for _, dt := range availableDatetimes {

		if !checkIfAllForecastsHoursAreAvailable(dt) {
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

		for _, hour := range availableHours() {
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
	filename, err := getSingleForecast(dt, hour)
	if err != nil {
		return "", err
	}

	fmt.Println("Extracting GRIB data...")
	allPoints, err := extractGribData(filename)
	if err != nil {
		log.Fatal("Error extracting GRIB data:", err)
	}

	fmt.Printf("Total points extracted: %d\n", len(allPoints))

	polygon := []Point{
		{Lat: 39.7153328, Lon: 1.1861908},
		{Lat: 39.7097536, Lon: 0.3860986},
		{Lat: 39.7049828, Lon: -1.2260914},
		{Lat: 37.8525431, Lon: -1.2438369},
		{Lat: 37.8358186, Lon: 1.1625552},
	}

	fmt.Println("Filtering points within polygon...")
	pointsInPolygon := filterPointsByPolygon(allPoints, polygon)
	fmt.Printf("Points within polygon: %d\n", len(pointsInPolygon))

	fmt.Println("\nSample of filtered points:")
	for i, point := range pointsInPolygon {
		fmt.Printf("Point %d: Lat=%.6f, Lon=%.6f, Value=%.6f\n",
			i+1, point.Lat, point.Lon, point.Value)
	}

	if len(pointsInPolygon) > 0 {
		var sum, min, max float64
		min = pointsInPolygon[0].Value
		max = pointsInPolygon[0].Value

		for _, point := range pointsInPolygon {
			sum += point.Value
			if point.Value < min {
				min = point.Value
			}
			if point.Value > max {
				max = point.Value
			}
		}

		average := sum / float64(len(pointsInPolygon))
		fmt.Printf("\nStatistics for filtered points:\n")
		fmt.Printf("Count: %d\n", len(pointsInPolygon))
		fmt.Printf("Average: %.6f\n", average)
		fmt.Printf("Min: %.6f\n", min)
		fmt.Printf("Max: %.6f\n", max)
	}

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