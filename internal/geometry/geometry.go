package geometry

import "math"

const (
	EARTH_RADIUS_KM = 6371
)

var POLYGON = []Point{
	{Lat: 39.7153328, Lon: 1.1861908},
	{Lat: 39.7097536, Lon: 0.3860986},
	{Lat: 39.7049828, Lon: -1.2260914},
	{Lat: 37.8525431, Lon: -1.2438369},
	{Lat: 37.8358186, Lon: 1.1625552},
}

type GeoPoint struct {
	Lat   float64
	Lon   float64
	Value float64
}

type Point struct {
	Lat float64
	Lon float64
}

func IsPointInPolygon(point Point, polygon []Point) bool {
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

func FilterPointsByPolygon(points []GeoPoint, polygon []Point) []GeoPoint {
	var filtered []GeoPoint

	for _, point := range points {
		if IsPointInPolygon(Point{Lat: point.Lat, Lon: point.Lon}, polygon) {
			filtered = append(filtered, point)
		}
	}

	return filtered
}