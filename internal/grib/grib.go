package grib

/*
#cgo pkg-config: eccodes
#include <eccodes.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"unsafe"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
)


func checkErr(msg string, err C.int) {
	if err != 0 {
		log.Fatalf("%s: error code %d", msg, err)
	}
}

func ExtractGribData(filename string) ([]geometry.GeoPoint, error) {
	file, err := os.Open(filename)
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

	// Process all messages in the GRIB file
	return extractAllMessages(buf)
}

func extractAllMessages(buf []byte) ([]geometry.GeoPoint, error) {
	var allPoints []geometry.GeoPoint
	offset := 0
	messageCount := 0
	
	for offset < len(buf) {
		cBuf := unsafe.Pointer(&buf[offset])
		size := C.size_t(len(buf) - offset)
		
		var errCode C.int
		var handle *C.codes_handle
		
		handle = C.codes_handle_new_from_message(nil, cBuf, size)
		if handle == nil {
			break // No more messages
		}
		
		messageCount++
		
		// Check if this message matches our criteria
		if shouldProcessMessage(handle) {
			points, err := extractGribDataFromHandle(handle)
			if err != nil {
				C.codes_handle_delete(handle)
				return nil, err
			}
			allPoints = append(allPoints, points...)
		}
		
		// Move to next message
		var messageSize C.long
		errCode = C.codes_get_long(handle, C.CString("totalLength"), &messageSize)
		C.codes_handle_delete(handle)
		
		if errCode != 0 {
			break
		}
		
		offset += int(messageSize)
	}
	
	return allPoints, nil
}

func shouldProcessMessage(handle *C.codes_handle) bool {
	// Helper function to get string parameter
	getString := func(key string) string {
		var sLen C.size_t = 64
		s := (*C.char)(C.malloc(sLen))
		defer C.free(unsafe.Pointer(s))
		
		errCode := C.codes_get_string(handle, C.CString(key), s, &sLen)
		if errCode == 0 {
			return C.GoStringN(s, C.int(sLen))
		}
		return ""
	}
	
	// Check for variable='tirf' or related thermal infrared variables
	variableKeys := []string{"shortName", "paramId", "variable", "cfVarName", "name"}
	variableFound := false
	
	for _, key := range variableKeys {
		value := getString(key)
		// Check for exact match or related thermal infrared terms
		if strings.Contains(strings.ToLower(value), "tirf") {
			variableFound = true
			break
		}
	}
	
	// Only process messages that contain 'tirf' specifically
	if variableFound {
		// Don't process all messages, only look for tirf
		return true
	}

	return false
}

func listGribParameters(handle *C.codes_handle) {
	fmt.Println("Available GRIB parameters:")
	
	// Common parameters to check
	parameters := []string{
		"shortName", "name", "units", "paramId", "centre", "subCentre",
		"generatingProcessIdentifier", "typeOfLevel", "level", "package",
		"paquet", "productDefinitionTemplateNumber", "cfVarName", "variable",
		"discipline", "parameterCategory", "parameterNumber",
	}
	
	for _, param := range parameters {
		var sLen C.size_t = 256
		s := (*C.char)(C.malloc(sLen))
		defer C.free(unsafe.Pointer(s))
		
		errCode := C.codes_get_string(handle, C.CString(param), s, &sLen)
		if errCode == 0 {
			value := C.GoStringN(s, C.int(sLen))
			fmt.Printf("  %s: %s\n", param, value)
		} else {
			// Try as long integer
			var longVal C.long
			errCode = C.codes_get_long(handle, C.CString(param), &longVal)
			if errCode == 0 {
				fmt.Printf("  %s: %d\n", param, longVal)
			}
		}
	}
	fmt.Println("---")
}

func extractGribDataFromHandle(handle *C.codes_handle) ([]geometry.GeoPoint, error) {
	var numberOfPoints C.long
	errCode := C.codes_get_long(handle, C.CString("numberOfPoints"), &numberOfPoints)
	checkErr("getting numberOfPoints", errCode)

	fmt.Printf("Number of points in this message: %d\n", numberOfPoints)

	var err C.int
	iter := C.codes_grib_iterator_new(handle, 0, &err)
	if iter == nil {
		return nil, fmt.Errorf("failed to create iterator: error code %d", err)
	}
	defer C.codes_grib_iterator_delete(iter)

	var points []geometry.GeoPoint
	var lat, lon, value C.double

	for C.codes_grib_iterator_next(iter, &lat, &lon, &value) == 1 {
		points = append(points, geometry.GeoPoint{
			Lat:   float64(lat),
			Lon:   float64(lon),
			Value: float64(value),
		})
	}

	return points, nil
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