package forecast

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"math"
	"sync"
	"time"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/grib"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/storage"
)

// Constants
const (
	FORECAST_HOURS = 51
)

func GetAvailableRunDates() []string {
	availableRunDates := []string{}
	now := time.Now()
	currentHour := now.Hour()
	yesterday := now.AddDate(0, 0, -1)

	runs := []int{21, 18, 15, 12, 9, 6, 3, 0}
	
	// Add available runs from current day
	for _, run := range runs {
		if run <= currentHour {
			availableRunDates = append(availableRunDates,
				now.Format("2006-01-02")+fmt.Sprintf("T%02d:00:00Z", run))
		}
	}

	// Add all runs from previous day (in early morning only runs from past day are available)
	for _, run := range runs {
		availableRunDates = append(availableRunDates,
			yesterday.Format("2006-01-02")+fmt.Sprintf("T%02d:00:00Z", run))
	}

	return availableRunDates
}

func CheckIfAllForecastsHoursAreAvailable(dt string) bool {
	hours := GetAvailableHours(dt)

	resultChans := make([]chan bool, len(hours))
	for i, hour := range hours {
		resultChans[i] = make(chan bool, 1)
		go func(h string, ch chan bool) {
			url := fmt.Sprintf("https://object.files.data.gouv.fr/meteofrance-pnt/pnt/%s/arome/001/SP2/arome__001__SP2__%sH__%s.grib2", dt, h, dt)
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

func GetSingleForecast(dt string, hour string) (string, error) {
	url := fmt.Sprintf("https://object.files.data.gouv.fr/meteofrance-pnt/pnt/%s/arome/001/SP2/arome__001__SP2__%sH__%s.grib2", dt, hour, dt)
	utils.Log("Downloading " + url)

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

	grib2file := fmt.Sprintf("./tmp/file_%s_%s.grib2", dt, hour)
	err = os.WriteFile(grib2file, body, 0644)
	if err != nil {
		return "", err
	}

	return grib2file, nil
}

func GetAvailableHours(dt string) []string {
	hours := make([]string, FORECAST_HOURS)

	for i := 0; i < FORECAST_HOURS; i++ {
		hours[i] = fmt.Sprintf("%02d", i + 1)
	}

	return hours
}

func ProcessSingleForecast(dt string, hour string) (string, error) {
	filename, err := GetSingleForecast(dt, hour)
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

			allData = append(allData, []float64{math.Round(point.Lon*1000)/1000, math.Round(point.Lat*1000)/1000, math.Round(value*100)/100})
		}
	}

	storage.Save(allData, hour, dt)

	return "", nil
}

func StartFetching() {
	availableRuns := GetAvailableRunDates()

	for _, run := range availableRuns {

		if !CheckIfAllForecastsHoursAreAvailable(run) {
			continue
		}

		if storage.IsUpToDate(run) {
			utils.Log("Forecast already downloaded, skipping " + run)
			time.Sleep(60 * time.Second)
			break
		}

		utils.Log("Forecast found for " + run)

		var (
			wg sync.WaitGroup
		)

		for _, hour := range GetAvailableHours(run) {
			wg.Add(1)
			go func(h string) {
				defer wg.Done()
				ProcessSingleForecast(run, h)
			}(hour)
		}

		wg.Wait()

		storage.RollOut()
		utils.CleanUpFiles()

		break
	}

	StartFetching()	
}