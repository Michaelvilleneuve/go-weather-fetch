package main

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
	"time"
	"unsafe"
)


func checkErr(msg string, err C.int) {
	if err != 0 {
		log.Fatalf("%s: error code %d", msg, err)
	}
}

func extractGribData(filename string) ([]GeoPoint, error) {
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

	return extractGribDataFromHandle(handle)
}

func extractGribDataFromHandle(handle *C.codes_handle) ([]GeoPoint, error) {
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