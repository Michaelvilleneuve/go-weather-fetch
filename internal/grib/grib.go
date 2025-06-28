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
	"unsafe"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
)


func checkErr(msg string, err C.int) {
	if err != 0 {
		log.Fatalf("%s: error code %d", msg, err)
	}
}

func ExtractGribData(filename string, fields []string) (map[string][]geometry.GeoPoint, error) {
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
	return extractAllMessages(buf, fields)
}

func extractAllMessages(buf []byte, fields []string) (map[string][]geometry.GeoPoint, error) {
	pointsByField := make(map[string][]geometry.GeoPoint)
	for _, field := range fields {
		pointsByField[field] = []geometry.GeoPoint{}
	}

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
		
		for _, field := range fields {
			if shouldProcessMessage(handle, field) {
				points, err := extractGribDataFromHandle(handle, field)
				if err != nil {
					C.codes_handle_delete(handle)
					return nil, err
				}
				pointsByField[field] = append(pointsByField[field], points...)
			}
		}
		
		var messageSize C.long
		errCode = C.codes_get_long(handle, C.CString("totalLength"), &messageSize)
		C.codes_handle_delete(handle)
		
		if errCode != 0 {
			break
		}
		
		offset += int(messageSize)
	}
	
	return pointsByField, nil
}

func shouldProcessMessage(handle *C.codes_handle, field string) bool {
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
	
	variableKeys := []string{"shortName", "paramId", "variable", "cfVarName", "name"}
	variableFound := false
	
	for _, key := range variableKeys {
		value := getString(key)
		if strings.Contains(strings.ToLower(value), field) {
			variableFound = true
			break
		}
	}
	
	return variableFound
}

func extractGribDataFromHandle(handle *C.codes_handle, field string) ([]geometry.GeoPoint, error) {
	var numberOfPoints C.long
	errCode := C.codes_get_long(handle, C.CString("numberOfPoints"), &numberOfPoints)
	checkErr("getting numberOfPoints", errCode)

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
