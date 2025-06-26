package fieldshandler

import (
	"fmt"
	"math"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
)

// comfortIndex calculates a comfort index on a scale of 1 to 10.
// This index is based on an apparent temperature calculation that considers
// temperature (t2m in Kelvin), wind speed (from u10, v10 in m/s), and
// relative humidity (r2 in %).
// The apparent temperature formula is adapted from Australia's Bureau of Meteorology.
// https://en.wikipedia.org/wiki/Apparent_temperature
//
// The resulting comfort index provides a normalized value where 1 is extremely cold
// and 10 is extremely hot.
func comfortIndex(t2m, u10, v10, r2 float64) float64 {
	// Validate input ranges
	if t2m < 200 || t2m > 350 || r2 < 0 || r2 > 100 || math.Abs(u10) > 100 || math.Abs(v10) > 100 {
		// Return a neutral comfort index if inputs are out of a reasonable range
		return 5.0
	}

	tC := t2m - 273.15
	windSpeed := math.Hypot(u10, v10)
	rh := r2

	// Calculate water vapour pressure (e) in hPa
	e := (rh / 100) * 6.105 * math.Exp(17.27*tC/(237.7+tC))

	// Calculate Apparent Temperature (AT) in Celsius
	// Formula from Australia's Bureau of Meteorology.
	at := tC + 0.33*e - 0.70*windSpeed - 4.00

	if math.IsNaN(at) || math.IsInf(at, 0) {
		return 5.0 // Return neutral for invalid calculation
	}

	// Convert AT to a comfort index from 1 (very cold) to 10 (very hot).
	// We map the AT range of -20°C to 50°C to the index range 1-10.
	const minAT, maxAT = -20.0, 50.0
	const minIndex, maxIndex = 1.0, 10.0

	if at <= minAT {
		return minIndex
	}
	if at >= maxAT {
		return maxIndex
	}

	// Linear interpolation
	index := minIndex + (at-minAT)*(maxIndex-minIndex)/(maxAT-minAT)

	return index
}

func ProcessComfortIndex(pointsByField map[string][]geometry.GeoPoint) map[string]geometry.GeoPoint {
	weatherDataMap := make(map[string]map[string]float64)
	coordinateMap := make(map[string]geometry.GeoPoint)

	// Collect weather data by coordinate - only store valid values
	for fieldName, points := range pointsByField {
		for _, point := range points {
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

	// Calculate comfort index for each coordinate
	for coordKey, weatherData := range weatherDataMap {
		t2m, hasT2m := weatherData["t2m"] // Temperature in Kelvin
		u10, hasU10 := weatherData["u10"] // U-component of wind at 10m (m/s)
		v10, hasV10 := weatherData["v10"] // V-component of wind at 10m (m/s)
		r2, hasR2 := weatherData["r2"]   // Relative humidity at 2m (%)

		// Only calculate if we have all required fields
		if hasT2m && hasU10 && hasV10 && hasR2 {
			comfort := comfortIndex(t2m, u10, v10, r2)

			// Parse coordinates from key
			var lon, lat float64
			fmt.Sscanf(coordKey, "%f,%f", &lon, &lat)

			coordinateMap[coordKey] = geometry.GeoPoint{
				Lat:   lat,
				Lon:   lon,
				Value: comfort,
			}
		}
	}

	return coordinateMap
} 