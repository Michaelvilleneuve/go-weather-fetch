package forecast

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/forecast/fieldshandler"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/grib"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/storage"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
)

type ForecastGroup struct {
	CommonName string
	Fields []string
}

type ForecastPackage struct {
	Package string
	Forecasts []ForecastGroup
}

const (
	FORECAST_HOURS = 51
)

var FORECAST_PACKAGES = []ForecastPackage{
	{
		Package: "SP2",
		Forecasts: []ForecastGroup{
			{CommonName: "rainfall_accumulation", Fields: []string{"tirf"}},
			{CommonName: "cloud_cover", Fields: []string{"lcc", "mcc", "hcc"}},
		},
	},
	{
		Package: "SP1",
		Forecasts: []ForecastGroup{
			{CommonName: "humidity", Fields: []string{"r2"}},
			{CommonName: "temperature", Fields: []string{"t2m"}},
			{CommonName: "feels_like_temperature", Fields: []string{"r2", "t2m", "u10", "v10"}},
		},
	},
}

func StartFetching() {
	var wg sync.WaitGroup
	
	for _, forecastPackage := range FORECAST_PACKAGES {
		wg.Add(1)
		go func(fp ForecastPackage) {
			defer wg.Done()
			processForecastPackage(fp)
		}(forecastPackage)
	}
	
	wg.Wait()

	StartFetching()
}

// Package is like SP1 or SP2 from méteo-france
// Each package is stored in a separate folder in data.gouv.fr
// So we download every hour of every package
func processForecastPackage(forecastPackage ForecastPackage) {
	run := getLatestCompleteRun(forecastPackage)

	if storage.IsUpToDate(forecastPackage.Package, run) {
		utils.Log("Forecast already downloaded, skipping " + run)
		time.Sleep(60 * time.Second)
		return
	}

	utils.Log("Forecast found for package " + forecastPackage.Package + " run: " + run)

	// Process each hour from 1 to 51
	for _, hour := range getAvailableHours() {
		filename, err := downloadPackage(forecastPackage.Package, run, hour)
		if err != nil {
			utils.Log("Error getting single forecast: " + err.Error())
			return
		}

		utils.Log("Forecast retrieved for " + run + " " + hour)

		// Now we process each param (temperature, humidity) of a given package
		processForecastGroup(filename, forecastPackage, run, hour)
	}

	// Extract common names from forecast groups
	commonNames := make([]string, len(forecastPackage.Forecasts))
	for i, fg := range forecastPackage.Forecasts {
		commonNames[i] = fg.CommonName
	}
	
	storage.RollOut(forecastPackage.Package, commonNames)
}

func processForecastGroup(filename string, forecastPackage ForecastPackage, run string, hour string) {
	for _, forecastGroup := range forecastPackage.Forecasts {
		ProcessSingleForecast(filename, forecastGroup.CommonName, forecastGroup.Fields, run, hour)
	}
}

func ProcessSingleForecast(filename string, commonName string, fields []string, dt string, hour string) (string, error) {
	pointsByField, err := grib.ExtractGribData(filename, fields)
	if err != nil {
		utils.Log("Error extracting GRIB data: " + err.Error())
		return "", err
	}

	// Process data based on forecast type
	var coordinateMap map[string]geometry.GeoPoint
	
	switch commonName {
	case "cloud_cover":
		coordinateMap = fieldshandler.ProcessCloudCover(pointsByField)
	case "feels_like_temperature":
		coordinateMap = fieldshandler.ProcessFeelsLikeTemperature(pointsByField)
	default:
		coordinateMap = fieldshandler.ProcessDefaultForecast(pointsByField)
	}
	
	// Convert coordinate map to output format
	allData := [][]float64{}
	for _, point := range coordinateMap {
		allData = append(allData, []float64{point.Lon, point.Lat, math.Round(point.Value*100)/100})
	}

	storage.Save(allData, commonName, hour, dt)

	return "", nil
}

func getAvailableRunDates() []string {
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

func allForecastsHoursAreAvailable(packageName string, dt string) bool {
	hours := getAvailableHours()

	resultChans := make([]chan bool, len(hours))
	for i, hour := range hours {
		resultChans[i] = make(chan bool, 1)
		go func(h string, ch chan bool) {
			url := fmt.Sprintf("https://object.files.data.gouv.fr/meteofrance-pnt/pnt/%s/arome/001/%s/arome__001__%s__%sH__%s.grib2", dt, packageName, packageName, h, dt)
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

func downloadPackage(packageName string, dt string, hour string) (string, error) {
	url := fmt.Sprintf("https://object.files.data.gouv.fr/meteofrance-pnt/pnt/%s/arome/001/%s/arome__001__%s__%sH__%s.grib2", dt, packageName, packageName, hour, dt)
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

	grib2file := fmt.Sprintf("./tmp/file_%s_%s_%s.grib2", packageName, dt, hour)
	err = os.WriteFile(grib2file, body, 0644)
	if err != nil {
		return "", err
	}

	return grib2file, nil
}

func getAvailableHours() []string {
	hours := make([]string, FORECAST_HOURS)

	for i := 0; i < FORECAST_HOURS; i++ {
		hours[i] = fmt.Sprintf("%02d", i + 1)
	}

	return hours
}

func getLatestCompleteRun(forecastPackage ForecastPackage) string {
	latestCompleteRun := ""
	for _, run := range getAvailableRunDates() {
		if allForecastsHoursAreAvailable(forecastPackage.Package, run) {
			latestCompleteRun = run
			break
		}
	}

	return latestCompleteRun
}


