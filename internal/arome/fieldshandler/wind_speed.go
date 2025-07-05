package fieldshandler

import (
	"fmt"
	"math"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
)

func ProcessWindSpeed(pointsByField map[string][]geometry.GeoPoint) map[string]geometry.GeoPoint {
	coordinateMap := make(map[string]geometry.GeoPoint)
	
	for _, points := range pointsByField {
		for _, point := range points {
			// Create a key from rounded coordinates for grouping
			coordKey := fmt.Sprintf("%.3f,%.3f", math.Round(point.Lon*1000)/1000, math.Round(point.Lat*1000)/1000)

			coordinateMap[coordKey] = geometry.GeoPoint{
				Lat:   math.Round(point.Lat*1000)/1000,
				Lon:   math.Round(point.Lon*1000)/1000,
				Value: point.Value, // Convert Kelvin to Celsius
			}
		}
	}
	
	return coordinateMap
} 