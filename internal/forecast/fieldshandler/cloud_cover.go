package fieldshandler

import (
	"fmt"
	"math"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/geometry"
)

// ProcessCloudCover calculates total cloud cover using the formula:
// tcc = lcc + mcc * (1 - lcc) + hcc * (1 - lcc) * (1 - mcc)
// where lcc = low cloud cover, mcc = medium cloud cover, hcc = high cloud cover
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
				// Convert from percentage (0-100) to fraction (0-1) if needed
				// AROME cloud cover values are typically in percentage format
				value = point.Value
				if value > 1.0 {
					value = value / 100.0 // Convert percentage to fraction
				}
			}
			cloudDataMap[coordKey][fieldName] = value
		}
	}
	
	// Calculate total cloud cover using the formula
	for coordKey, cloudData := range cloudDataMap {
		lcc := cloudData["lcc"]
		mcc := cloudData["mcc"] 
		hcc := cloudData["hcc"]
		
		// Ensure values are in valid range [0,1]
		lcc = math.Max(0, math.Min(1, lcc))
		mcc = math.Max(0, math.Min(1, mcc))
		hcc = math.Max(0, math.Min(1, hcc))
		
		// Apply the total cloud cover formula (result is fraction 0-1)
		tcc := lcc + mcc*(1-lcc) + hcc*(1-lcc)*(1-mcc)
		
		// Ensure result is in valid range and convert back to percentage for consistency
		tcc = math.Max(0, math.Min(1, tcc)) * 100.0
		
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
