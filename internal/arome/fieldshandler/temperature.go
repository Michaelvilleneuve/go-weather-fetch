package fieldshandler

import (
	"fmt"
	"math"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
)

// ProcessTemperatureForecast handles the temperature behavior of summing values from all fields
func ProcessTemperatureForecast(pointsByField map[string][]geometry.GeoPoint) map[string]geometry.GeoPoint {
	coordinateMap := make(map[string]geometry.GeoPoint)
	
	for _, points := range pointsByField {
		for _, point := range points {
			// Create a key from rounded coordinates for grouping
			coordKey := fmt.Sprintf("%.3f,%.3f", math.Round(point.Lon*1000)/1000, math.Round(point.Lat*1000)/1000)
			
			value := 0.0
			if point.Value < 9999 {
				value = point.Value
			}

			valueInCelsius := value - 273.15

			if (valueInCelsius < -70) {
				continue
			}

			coordinateMap[coordKey] = geometry.GeoPoint{
				Lat:   math.Round(point.Lat*1000)/1000,
				Lon:   math.Round(point.Lon*1000)/1000,
				Value: valueInCelsius, // Convert Kelvin to Celsius
			}
		}
	}
	
	return coordinateMap
} 