package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Constants
const (
	FORECAST_HOURS = 3
)

func getAvailableDatetimes() []string {
	availableDatetimes := []string{}
	now := time.Now()
	currentHour := now.Hour()

	runs := []int{21, 18, 15, 12, 9, 5, 3, 0}
	for _, run := range runs {
		if run <= currentHour {
			availableDatetimes = append(availableDatetimes,
				now.Format("2006-01-02")+fmt.Sprintf("T%02d:00:00Z", run))
		}
	}

	return availableDatetimes
}

func checkIfAllForecastsHoursAreAvailable(dt string) bool {
	hours := make([]string, FORECAST_HOURS)
	for i := 0; i < FORECAST_HOURS; i++ {
		hours[i] = fmt.Sprintf("%02d", i)
	}

	resultChans := make([]chan bool, len(hours))
	for i, hour := range hours {
		resultChans[i] = make(chan bool, 1)
		go func(h string, ch chan bool) {
			url := fmt.Sprintf("https://object.files.data.gouv.fr/meteofrance-pnt/pnt/%s/arome/001/HP1/arome__001__HP1__%sH__%s.grib2", dt, h, dt)
			response, err := http.Head(url)
			if err != nil {
				ch <- false
				return
			}
			ch <- response.StatusCode == 200
		}(hour, resultChans[i])
	}

	allAvailable := true
	for _, ch := range resultChans {
		if !<-ch {
			allAvailable = false
			break
		}
	}

	return allAvailable
}

func getSingleForecast(dt string, hour string) (string, error) {
	url := fmt.Sprintf("https://object.files.data.gouv.fr/meteofrance-pnt/pnt/%s/arome/001/HP1/arome__001__HP1__%sH__%s.grib2", dt, hour, dt)
	log.Printf("Downloading %s", url)

	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode == 404 {
		return "", fmt.Errorf("file not found: %s", url)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	log.Printf("Downloaded %d bytes", len(body))

	grib2file := fmt.Sprintf("./tmp/file_%s_%s.grib2", dt, hour)
	err = os.WriteFile(grib2file, body, 0644)
	if err != nil {
		return "", err
	}

	return grib2file, nil
}

func availableHours() []string {
	hours := make([]string, FORECAST_HOURS)

	for i := 0; i < FORECAST_HOURS; i++ {
		hours[i] = fmt.Sprintf("%02d", i)
	}

	return hours
}