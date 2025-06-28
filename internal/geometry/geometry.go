package geometry

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
)

func GetPolygon() ([]Point, error) {
	polygonEnv := os.Getenv("POLYGON_COORDINATES")

	if polygonEnv == "" {
		return []Point{}, fmt.Errorf("POLYGON_COORDINATES not set")
	}

	var polygon []Point
	if err := json.Unmarshal([]byte(polygonEnv), &polygon); err != nil {
		log.Printf("Error parsing POLYGON_COORDINATES: %v", err)
		return []Point{}, err
	}

	return polygon, nil
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

func IsPointInPolygon(point Point) bool {
	polygon, err := GetPolygon()

	if err != nil {
		return true
	}

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
