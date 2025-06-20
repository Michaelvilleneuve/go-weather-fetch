package fieldshandler

import (
	"fmt"
	"math"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
)

// ProcessDefaultForecast handles the default behavior of summing values from all fields
func ProcessDefaultForecast(pointsByField map[string][]geometry.GeoPoint) map[string]geometry.GeoPoint {
	coordinateMap := make(map[string]geometry.GeoPoint)
	
	for _, points := range pointsByField {
		for _, point := range points {
			if !geometry.IsPointInPolygon(geometry.Point{Lat: point.Lat, Lon: point.Lon}, geometry.POLYGON) {
				continue
			}
			
			// Create a key from rounded coordinates for grouping
			coordKey := fmt.Sprintf("%.3f,%.3f", math.Round(point.Lon*1000)/1000, math.Round(point.Lat*1000)/1000)
			
			value := 0.0
			if point.Value < 9999 {
				value = point.Value
			}
			
			if existingPoint, exists := coordinateMap[coordKey]; exists {
				// Sum the values if coordinate already exists
				existingPoint.Value += value
				coordinateMap[coordKey] = existingPoint
			} else {
				// Create new point entry
				coordinateMap[coordKey] = geometry.GeoPoint{
					Lat:   math.Round(point.Lat*1000)/1000,
					Lon:   math.Round(point.Lon*1000)/1000,
					Value: value,
				}
			}
		}
	}
	
	return coordinateMap
} 