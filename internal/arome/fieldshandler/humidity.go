package fieldshandler

import (
	"fmt"
	"math"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
)

// ProcessHumidityForecast handles the humidity behavior of summing values from all fields
func ProcessHumidityForecast(pointsByField map[string][]geometry.GeoPoint) map[string]geometry.GeoPoint {
	coordinateMap := make(map[string]geometry.GeoPoint)
	
	for _, points := range pointsByField {
		for _, point := range points {
			// Create a key from rounded coordinates for grouping
			coordKey := fmt.Sprintf("%.3f,%.3f", math.Round(point.Lon*1000)/1000, math.Round(point.Lat*1000)/1000)
			
			value := point.Value

			if value == 0 || value == 9999 {
				continue
			}
			
			coordinateMap[coordKey] = geometry.GeoPoint{
				Lat:   math.Round(point.Lat*1000)/1000,
				Lon:   math.Round(point.Lon*1000)/1000,
				Value: value,
			}
		}
	}
	
	return coordinateMap
} 