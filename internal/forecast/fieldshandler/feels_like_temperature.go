package fieldshandler

import (
	"fmt"
	"math"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
)

// FeelsLikeTemperature calculates the "feels like" temperature in Celsius
// from t2m (Kelvin), u10 and v10 (m/s), r2 (relative humidity %)
func feelsLikeTemperature(t2m, u10, v10, r2 float64) float64 {
	// Validate input ranges first
	if t2m < 200 || t2m > 350 {  // Reasonable temperature range in Kelvin (~ -73°C to 77°C)
		return t2m - 273.15  // Just return actual temperature if outside reasonable range
	}
	if r2 < 0 || r2 > 100 {  // Humidity should be 0-100%
		return t2m - 273.15
	}
	if math.Abs(u10) > 100 || math.Abs(v10) > 100 {  // Extreme wind speeds
		return t2m - 273.15
	}

	// Convert temperature from Kelvin to Celsius
	tC := t2m - 273.15

	// Compute wind speed in m/s and km/h
	windSpeed := math.Hypot(u10, v10)
	windSpeedKmh := windSpeed * 3.6

	// Apply Wind Chill formula
	if tC < 10.0 && windSpeed > 1.3 {
		result := 13.12 +
			0.6215*tC -
			11.37*math.Pow(windSpeedKmh, 0.16) +
			0.3965*tC*math.Pow(windSpeedKmh, 0.16)
		
		// Validate result
		if math.IsNaN(result) || math.IsInf(result, 0) || result < -100 || result > 100 {
			return tC  // Return actual temperature if calculation is invalid
		}
		return result
	}

	// Apply Heat Index formula
	if tC >= 27.0 && r2 >= 40.0 {
		T := tC
		RH := r2
		result := -8.7847 +
			1.6114*T +
			2.3385*RH -
			0.1461*T*RH -
			0.0123*T*T -
			0.0164*RH*RH +
			0.0022*T*T*RH +
			0.0007*T*RH*RH -
			0.0003*T*T*RH*RH
		
		// Validate result
		if math.IsNaN(result) || math.IsInf(result, 0) || result < -100 || result > 100 {
			return tC  // Return actual temperature if calculation is invalid
		}
		return result
	}

	// Otherwise, just return actual temperature
	return tC
}

func ProcessFeelsLikeTemperature(pointsByField map[string][]geometry.GeoPoint) map[string]geometry.GeoPoint {
	weatherDataMap := make(map[string]map[string]float64)
	coordinateMap := make(map[string]geometry.GeoPoint)
	
	// Collect weather data by coordinate - only store valid values
	for fieldName, points := range pointsByField {
		for _, point := range points {
			if !geometry.IsPointInPolygon(geometry.Point{Lat: point.Lat, Lon: point.Lon}, geometry.POLYGON) {
				continue
			}
			
			// Only store values that are valid (< 9999)
			if point.Value < 9999 {
				coordKey := fmt.Sprintf("%.3f,%.3f", math.Round(point.Lon*1000)/1000, math.Round(point.Lat*1000)/1000)
				
				if weatherDataMap[coordKey] == nil {
					weatherDataMap[coordKey] = make(map[string]float64)
				}
				
				weatherDataMap[coordKey][fieldName] = point.Value
			}
		}
	}
	
	// Calculate feels-like temperature for each coordinate
	for coordKey, weatherData := range weatherDataMap {
		t2m, hasT2m := weatherData["t2m"]   // Temperature in Kelvin
		u10, hasU10 := weatherData["u10"]   // U-component of wind at 10m (m/s)
		v10, hasV10 := weatherData["v10"]   // V-component of wind at 10m (m/s)
		r2, hasR2 := weatherData["r2"]     // Relative humidity at 2m (%)
		
		// Only calculate if we have all required fields
		if hasT2m && hasU10 && hasV10 && hasR2 {
			feelsLike := feelsLikeTemperature(t2m, u10, v10, r2)
			
			// Parse coordinates from key
			var lon, lat float64
			fmt.Sscanf(coordKey, "%f,%f", &lon, &lat)
			
			coordinateMap[coordKey] = geometry.GeoPoint{
				Lat:   lat,
				Lon:   lon,
				Value: feelsLike,
			}
		}
	}
	
	return coordinateMap
} 