package main

/*
#cgo pkg-config: eccodes
#include <eccodes.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"
	"unsafe"
)

type GeoPoint struct {
	Lat   float64
	Lon   float64
	Value float64
}

type Point struct {
	Lat float64
	Lon float64
}

func checkErr(msg string, err C.int) {
	if err != 0 {
		log.Fatalf("%s: error code %d", msg, err)
	}
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func isPointInPolygon(point Point, polygon []Point) bool {
	x, y := point.Lon, point.Lat
	n := len(polygon)
	inside := false

	p1x, p1y := polygon[0].Lon, polygon[0].Lat
	for i := 1; i <= n; i++ {
		p2x, p2y := polygon[i%n].Lon, polygon[i%n].Lat
		if y > math.Min(p1y, p2y) {
			if y <= math.Max(p1y, p2y) {
				if x <= math.Max(p1x, p2x) {
					var xinters float64
					if p1y != p2y {
						xinters = (y-p1y)*(p2x-p1x)/(p2y-p1y) + p1x
					}
					if p1x == p2x || x <= xinters {
						inside = !inside
					}
				}
			}
		}
		p1x, p1y = p2x, p2y
	}

	return inside
}

func extractGribData(handle *C.codes_handle) ([]GeoPoint, error) {
	var numberOfPoints C.long
	errCode := C.codes_get_long(handle, C.CString("numberOfPoints"), &numberOfPoints)
	checkErr("getting numberOfPoints", errCode)

	fmt.Printf("Total number of points: %d\n", numberOfPoints)

	var err C.int
	iter := C.codes_grib_iterator_new(handle, 0, &err)
	if iter == nil {
		return nil, fmt.Errorf("failed to create iterator: error code %d", err)
	}
	defer C.codes_grib_iterator_delete(iter)

	var points []GeoPoint
	var lat, lon, value C.double

	for C.codes_grib_iterator_next(iter, &lat, &lon, &value) == 1 {
		points = append(points, GeoPoint{
			Lat:   float64(lat),
			Lon:   float64(lon),
			Value: float64(value),
		})
	}

	return points, nil
}

func filterPointsByRadius(points []GeoPoint, centerLat, centerLon, radiusKm float64) []GeoPoint {
	var filtered []GeoPoint

	for _, point := range points {
		distance := haversineDistance(centerLat, centerLon, point.Lat, point.Lon)
		if distance <= radiusKm {
			filtered = append(filtered, point)
		}
	}

	return filtered
}

func filterPointsByPolygon(points []GeoPoint, polygon []Point) []GeoPoint {
	var filtered []GeoPoint

	for _, point := range points {
		if isPointInPolygon(Point{Lat: point.Lat, Lon: point.Lon}, polygon) {
			filtered = append(filtered, point)
		}
	}

	return filtered
}

func getTimestamp(handle *C.codes_handle) (time.Time, error) {
	var year, month, day, hour, minute, second C.long

	errCode := C.codes_get_long(handle, C.CString("year"), &year)
	checkErr("getting year", errCode)

	errCode = C.codes_get_long(handle, C.CString("month"), &month)
	checkErr("getting month", errCode)

	errCode = C.codes_get_long(handle, C.CString("day"), &day)
	checkErr("getting day", errCode)

	errCode = C.codes_get_long(handle, C.CString("hour"), &hour)
	checkErr("getting hour", errCode)

	errCode = C.codes_get_long(handle, C.CString("minute"), &minute)
	checkErr("getting minute", errCode)

	errCode = C.codes_get_long(handle, C.CString("second"), &second)
	if errCode != 0 {
		second = 0
	}

	timestamp := time.Date(int(year), time.Month(month), int(day),
		int(hour), int(minute), int(second), 0, time.UTC)

	return timestamp, nil
}

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
	hours := make([]string, 52)
	for i := 0; i <= 51; i++ {
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

func main() {
	hours := make([]string, 52)

	for i := 0; i <= 51; i++ {
		hours[i] = fmt.Sprintf("%02d", i)
	}

	availableDatetimes := getAvailableDatetimes()

	for _, dt := range availableDatetimes {
		if !checkIfAllForecastsHoursAreAvailable(dt) {
			log.Printf("No forecast yet for %s", dt)
			continue
		}

		err := os.WriteFile("last_download.txt", []byte(dt), 0644)

		if err != nil {
			log.Printf("Error writing last_download.txt: %v", err)
		}

		var (
			wg sync.WaitGroup
		)

		for _, hour := range hours {
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

func cleanUpFiles() {
	files, err := os.ReadDir("./tmp")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		os.Remove(file.Name())
	}
}

func processSingleForecast(dt string, hour string) (string, error) {
	grib2file, err := getSingleForecast(dt, hour)

	file, err := os.Open(grib2file)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, stat.Size())
	_, err = file.Read(buf)
	if err != nil {
		log.Fatal(err)
	}

	cBuf := unsafe.Pointer(&buf[0])
	size := C.size_t(len(buf))

	var errCode C.int
	var handle *C.codes_handle

	handle = C.codes_handle_new_from_message(nil, cBuf, size)
	if handle == nil {
		log.Fatal("Failed to create handle from GRIB message")
	}
	defer C.codes_handle_delete(handle)

	var sLen C.size_t = 64
	s := (*C.char)(C.malloc(sLen))
	defer C.free(unsafe.Pointer(s))

	errCode = C.codes_get_string(handle, C.CString("shortName"), s, &sLen)
	checkErr("getting shortName", errCode)
	shortName := C.GoStringN(s, C.int(sLen))
	fmt.Println("shortName:", shortName)

	timestamp, err := getTimestamp(handle)
	if err != nil {
		log.Printf("Error getting timestamp: %v", err)
	} else {
		fmt.Printf("Timestamp: %s\n", timestamp.Format("2006-01-02 15:04:05"))
	}

	fmt.Println("Extracting GRIB data...")
	allPoints, err := extractGribData(handle)
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

	centerLat := 38.74684482459309
	centerLon := -0.0470862108957589
	radiusKm := 130.0

	fmt.Printf("Filtering points within %.1f km radius...\n", radiusKm)
	pointsInRadius := filterPointsByRadius(allPoints, centerLat, centerLon, radiusKm)
	fmt.Printf("Points within radius: %d\n", len(pointsInRadius))

	fmt.Println("Filtering points within polygon...")
	pointsInPolygon := filterPointsByPolygon(allPoints, polygon)
	fmt.Printf("Points within polygon: %d\n", len(pointsInPolygon))

	fmt.Println("Filtering points within both radius and polygon...")
	var finalPoints []GeoPoint
	for _, point := range pointsInRadius {
		if isPointInPolygon(Point{Lat: point.Lat, Lon: point.Lon}, polygon) {
			finalPoints = append(finalPoints, point)
		}
	}
	fmt.Printf("Final filtered points: %d\n", len(finalPoints))

	fmt.Println("\nSample of filtered points:")
	for i, point := range finalPoints {
			break
		}
		fmt.Printf("Point %d: Lat=%.6f, Lon=%.6f, Value=%.6f\n",
			i+1, point.Lat, point.Lon, point.Value)
	}

	if len(finalPoints) > 0 {
		var sum, min, max float64
		min = finalPoints[0].Value
		max = finalPoints[0].Value

		for _, point := range finalPoints {
			sum += point.Value
			if point.Value < min {
				min = point.Value
			}
			if point.Value > max {
				max = point.Value
			}
		}

		average := sum / float64(len(finalPoints))
		fmt.Printf("\nStatistics for filtered points:\n")
		fmt.Printf("Count: %d\n", len(finalPoints))
		fmt.Printf("Average: %.6f\n", average)
		fmt.Printf("Min: %.6f\n", min)
		fmt.Printf("Max: %.6f\n", max)
	}
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
