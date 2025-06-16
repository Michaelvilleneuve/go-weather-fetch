package main

import "math"

// Constants
const (
	EARTH_RADIUS_KM = 6371 // Earth radius in kilometers
)

type GeoPoint struct {
	Lat   float64
	Lon   float64
	Value float64
}

type Point struct {
	Lat float64
	Lon float64
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EARTH_RADIUS_KM * c
} 

func isPointInPolygon(point Point, polygon []Point) bool {
	x, y := point.Lon, point.Lat
	n := len(polygon)
	inside := false

	p1x, p1y := polygon[0].Lon, polygon[0].Lat
	for i := 1; i <= n; i++ {
		p2x, p2y := polygon[i%n].Lon, polygon[i%n].Lat
		if y > math.Min(p1y, p2y) {
			if y <= math.Max(p1y, p2y) {
				if x <= math.Max(p1x, p2x) {
					var xinters float64
					if p1y != p2y {
						xinters = (y-p1y)*(p2x-p1x)/(p2y-p1y) + p1x
					}
					if p1x == p2x || x <= xinters {
						inside = !inside
					}
				}
			}
		}
		p1x, p1y = p2x, p2y
	}

	return inside
}


func filterPointsByRadius(points []GeoPoint, centerLat, centerLon, radiusKm float64) []GeoPoint {
	var filtered []GeoPoint

	for _, point := range points {
		distance := haversineDistance(centerLat, centerLon, point.Lat, point.Lon)
		if distance <= radiusKm {
			filtered = append(filtered, point)
		}
	}

	return filtered
}

func filterPointsByPolygon(points []GeoPoint, polygon []Point) []GeoPoint {
	var filtered []GeoPoint

	for _, point := range points {
		if isPointInPolygon(Point{Lat: point.Lat, Lon: point.Lon}, polygon) {
			filtered = append(filtered, point)
		}
	}

	return filtered
}