package fieldshandler

import (
	"fmt"
	"math"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
)

// cloudCover calculates total cloud cover from low, medium, and high cloud cover fractions.
// lcc, mcc, hcc are expected to be fractions (0-1).
// The formula used is: tcc = lcc + mcc * (1 - lcc) + hcc * (1 - lcc) * (1 - mcc)
// The result is the total cloud cover in percentage (0-100).
func cloudCover(lcc, mcc, hcc float64) float64 {
	// Ensure input values are fractions in the range [0,1]
	lcc = math.Max(0, math.Min(1, lcc))
	mcc = math.Max(0, math.Min(1, mcc))
	hcc = math.Max(0, math.Min(1, hcc))

	// Apply the total cloud cover formula
	tccFraction := lcc + mcc*(1-lcc) + hcc*(1-lcc)*(1-mcc)

	// Ensure result is in valid range [0,1] and convert to percentage
	tccPercentage := math.Max(0, math.Min(1, tccFraction)) * 100.0

	return tccPercentage
}

// ProcessCloudCover aggregates cloud cover data (lcc, mcc, hcc) for geographic points,
// calculates the total cloud cover for each point, and returns a map of points with their calculated cloud cover values.
func ProcessCloudCover(pointsByField map[string][]geometry.GeoPoint) map[string]geometry.GeoPoint {
	cloudDataMap := make(map[string]map[string]float64)
	coordinateMap := make(map[string]geometry.GeoPoint)

	// Collect cloud data by coordinate
	for fieldName, points := range pointsByField {
		for _, point := range points {
			if !geometry.IsPointInPolygon(geometry.Point{Lat: point.Lat, Lon: point.Lon}, geometry.POLYGON) {
				continue
			}

			coordKey := fmt.Sprintf("%.3f,%.3f", math.Round(point.Lon*1000)/1000, math.Round(point.Lat*1000)/1000)

			if cloudDataMap[coordKey] == nil {
				cloudDataMap[coordKey] = make(map[string]float64)
			}

			value := 0.0
			if point.Value < 9999 {
				// AROME cloud cover values are typically in percentage format (0-100),
				// but we handle fractional input (0-1) as well.
				value = point.Value
				if value > 1.0 {
					value = value / 100.0 // Convert percentage to fraction
				}
			}
			cloudDataMap[coordKey][fieldName] = value
		}
	}

	// Calculate total cloud cover using the extracted function
	for coordKey, cloudData := range cloudDataMap {
		lcc := cloudData["lcc"]
		mcc := cloudData["mcc"]
		hcc := cloudData["hcc"]

		// Calculate total cloud cover
		tcc := cloudCover(lcc, mcc, hcc)

		// Parse coordinates from key
		var lon, lat float64
		fmt.Sscanf(coordKey, "%f,%f", &lon, &lat)

		coordinateMap[coordKey] = geometry.GeoPoint{
			Lat:   lat,
			Lon:   lon,
			Value: tcc,
		}
	}

	return coordinateMap
}
