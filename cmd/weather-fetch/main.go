package main

import (
	"os"
	"sync"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/forecast"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/grib"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/api"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
)


func main() {
	utils.LoadEnv()
	availableDatetimes := forecast.GetAvailableRunDates()

	for _, dt := range availableDatetimes {

		if !forecast.CheckIfAllForecastsHoursAreAvailable(dt) {
			continue
		}

		utils.Log("Forecast found for " + dt)

		err := os.WriteFile("last_download.txt", []byte(dt), 0644)

		if err != nil {
			utils.Log("Error writing last_download.txt: " + err.Error())
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

		utils.CleanUpFiles()

		os.Exit(0)
	}
}

func processSingleForecast(dt string, hour string) (string, error) {
	filename, err := forecast.GetSingleForecast(dt, hour)
	if err != nil {
		return "", err
	}

	allPoints, err := grib.ExtractGribData(filename)
	if err != nil {
		utils.Log("Error extracting GRIB data: " + err.Error())
		return "", err
	}

	pointsInPolygon := geometry.FilterPointsByPolygon(allPoints, geometry.POLYGON)

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
