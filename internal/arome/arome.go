package arome

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/arome/fieldshandler"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/grib"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/storage"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
	"gopkg.in/yaml.v3"
)

const (
	DEFAULT_START_HOUR = 0
	DEFAULT_END_HOUR   = 51
)

type AromeField struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Unit string `yaml:"unit"`
	Description string `yaml:"description"`
}

type AromeLayer struct {
	CommonName string `yaml:"name"`
	Fields []AromeField `yaml:"fields"`
}

type AromePackage struct {
	Name string `yaml:"name"`
	Layers []AromeLayer `yaml:"layers"`
	Run *string
	Hour *string
	GribFileName *string
	ProcessedFiles []storage.ProcessedFile
}

type Arome struct {
	Packages []AromePackage `yaml:"packages"`
}

func (arome Arome) GetLayers() []AromeLayer {
	layers := []AromeLayer{}
	for _, aromePackage := range arome.Packages {
		layers = append(layers, aromePackage.Layers...)
	}
	return layers
}

func (arome Arome) GetLayerNames() []string {
	layers := []string{}
	for _, aromePackage := range arome.Packages {
		for _, aromeLayer := range aromePackage.Layers {
			layers = append(layers, aromeLayer.CommonName)
		}
	}
	return layers
}

func (aromePackage AromePackage) GetLayerNames() []string {
	layers := []string{}
	for _, aromeLayer := range aromePackage.Layers {
		layers = append(layers, aromeLayer.CommonName)
	}
	return layers
}

func (layer AromeLayer) GetFieldsNames() []string {
	fields := []string{}
	for _, field := range layer.Fields {
		fields = append(fields, field.Name)
	}
	return fields
}

func (aromePackage AromePackage) getLatestRun() string {
	latestRun := ""
	for _, run := range aromePackage.getAvailableRunDates() {
		if aromePackage.anyHourIsAvailable(run) {
			latestRun = run
			break
		}
	}

	return latestRun
}

func (aromePackage AromePackage) anyHourIsAvailable(dt string) bool {
	hours := aromePackage.getRunHoursToProcess()
	for _, hour := range hours {
		if aromePackage.isHourAvailable(dt, hour) {
			return true
		}
	}
	return false
}

func (aromePackage AromePackage) isHourAvailable(dt string, hour string) bool {
	url := fmt.Sprintf("https://object.files.data.gouv.fr/meteofrance-pnt/pnt/%s/arome/001/%s/arome__001__%s__%sH__%s.grib2", dt, aromePackage.Name, aromePackage.Name, hour, dt)
	response, err := http.Head(url)
	if err != nil {
		return false
	}
	return response.StatusCode == 200
}

func (aromePackage AromePackage) getRunHoursToProcess() []string {
	startHour := DEFAULT_START_HOUR
	endHour := DEFAULT_END_HOUR

	// Parse start hour from environment variable
	if startStr := os.Getenv("FORECAST_START_HOUR"); startStr != "" {
		if parsed, err := strconv.Atoi(startStr); err == nil && parsed >= 1 {
			startHour = parsed
		}
	}

	// Parse end hour from environment variable
	if endStr := os.Getenv("FORECAST_END_HOUR"); endStr != "" {
		if parsed, err := strconv.Atoi(endStr); err == nil && parsed >= startHour {
			endHour = parsed
		}
	}

	numHours := endHour - startHour + 1
	hours := make([]string, numHours)

	for i := 0; i < numHours; i++ {
		hours[i] = fmt.Sprintf("%02d", startHour + i)
	}

	return hours
}


func (aromePackage AromePackage) getAvailableRunDates() []string {
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

func (aromePackage AromePackage) ProcessLatestRun() {
	run := aromePackage.getLatestRun()
	
	for _, hour := range aromePackage.getRunHoursToProcess() {
		if aromePackage.AlreadyProcessed(run, hour) {
			utils.Log("Forecast already processed, skipping " + run + " " + hour)
			continue
		}

		if !aromePackage.isHourAvailable(run, hour) {
			utils.Log("Hour " + hour + " is not available, skipping")
			continue
		}

		downloadedPackage, err := aromePackage.downloadPackage(run, hour)
		if err != nil {
			utils.Log("Error downloading package: " + err.Error())
			continue
		}

		utils.Log("Forecast retrieved for " + run + " " + hour)

		downloadedPackage.processLayers()
		aromePackage.MarkAsProcessed(run, hour)

		for _, processedFile := range downloadedPackage.ProcessedFiles {
			go storage.RolloutRemotely(processedFile)
		}
		
		utils.Log("Forecast processed and rolled out for " + run + " " + hour)
	}
}


func (aromePackage AromePackage) AlreadyProcessed(run string, hour string) bool {
	_, err := os.Stat(fmt.Sprintf("tmp/%s_%s_%s.txt", aromePackage.Name, run, hour))
	return err == nil
}

func (aromePackage AromePackage) MarkAsProcessed(run string, hour string) {
	os.WriteFile(fmt.Sprintf("tmp/%s_%s_%s.txt", aromePackage.Name, run, hour), []byte(hour), 0644)
}


func (aromePackage AromePackage) downloadPackage(dt string, hour string) (AromePackage, error) {
	url := fmt.Sprintf("https://object.files.data.gouv.fr/meteofrance-pnt/pnt/%s/arome/001/%s/arome__001__%s__%sH__%s.grib2", dt, aromePackage.Name, aromePackage.Name, hour, dt)
	utils.Log("Downloading " + url)

	response, err := http.Get(url)
	if err != nil {
		return AromePackage{}, err
	}
	defer response.Body.Close()

	if response.StatusCode == 404 {
		return AromePackage{}, fmt.Errorf("file not found: %s", url)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return AromePackage{}, err
	}

	grib2file := fmt.Sprintf("./tmp/file_%s_%s_%s.grib2", aromePackage.Name, dt, hour)
	err = os.WriteFile(grib2file, body, 0644)
	if err != nil {
		return AromePackage{}, err
	}

	aromePackage.GribFileName = &grib2file
	aromePackage.Run = &dt
	aromePackage.Hour = &hour

	return aromePackage, nil
}

func (aromePackage *AromePackage) processLayers() {
	var wg sync.WaitGroup
	for _, aromeLayer := range aromePackage.Layers {
		wg.Add(1)
		go func(layer AromeLayer) {
			defer wg.Done()
			aromePackage.processLayer(layer)
		}(aromeLayer)
	}
	wg.Wait()
}



func (aromePackage *AromePackage) processLayer(layer AromeLayer) (string, error) {
	pointsByField, err := grib.ExtractGribData(*aromePackage.GribFileName, layer.GetFieldsNames())
	if err != nil {
		utils.Log("Error extracting GRIB data: " + err.Error())
		return "", err
	}

	// Process data based on forecast type
	var coordinateMap map[string]geometry.GeoPoint
	
	switch layer.CommonName {
	case "cloud_cover":
		coordinateMap = fieldshandler.ProcessCloudCover(pointsByField)
	case "comfort_index":
		coordinateMap = fieldshandler.ProcessComfortIndex(pointsByField)
	case "wind_speed":
		coordinateMap = fieldshandler.ProcessWindSpeed(pointsByField)
	case "temperature":
		coordinateMap = fieldshandler.ProcessTemperatureForecast(pointsByField)
	case "humidity":
		coordinateMap = fieldshandler.ProcessHumidityForecast(pointsByField)
	default:
		coordinateMap = fieldshandler.ProcessDefaultForecast(pointsByField)
	}
	
	// Convert coordinate map to output format
	allData := [][]float64{}
	for _, point := range coordinateMap {
		allData = append(allData, []float64{point.Lon, point.Lat, math.Round(point.Value*100)/100})
	}

	processedFile, _ := storage.Save(allData, storage.ProcessedFile{
		Model: "arome",
		Run: *aromePackage.Run,
		Layer: layer.CommonName,
		Hour: *aromePackage.Hour,
	})
	aromePackage.ProcessedFiles = append(aromePackage.ProcessedFiles, processedFile)

	return "", nil
}

func Configuration() Arome {
	yamlFile, err := os.ReadFile("config/arome.yml")
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	var config Arome
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("Error unmarshalling config file: %v", err)
	}

	return config
}
