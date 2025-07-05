package storage

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"sort"
	"strconv"
	"strings"
)

type ColorPoint struct {
	Value float64 `json:"volume"`
	Color string  `json:"color"`
}

type Palette struct {
	Name        string       `json:"name"`
	Icon        string       `json:"icon"`
	Unit        string       `json:"unit"`
	ShowPalette bool         `json:"show_palette"`
	Colors      []ColorPoint `json:"colors"`
}

var palettes = map[string]Palette{
	"comfort_index": {
		Name:        "comfort_index",
		Icon:        "temperature.svg",
		Unit:        "°C",
		ShowPalette: true,
		Colors: []ColorPoint{
			{Value: -40, Color: "#000080"},  // Très froid - bleu foncé
			{Value: -20, Color: "#0000FF"},  // Froid extrême - bleu
			{Value: -10, Color: "#4169E1"},  // Froid - bleu royal
			{Value: 0, Color: "#ADD8E6"},    // Frais - bleu clair
			{Value: 10, Color: "#90EE90"},   // Doux - vert clair
			{Value: 15, Color: "#FFFFE0"},   // Neutre - jaune très clair
			{Value: 20, Color: "#FFD700"},   // Agréable - jaune doré
			{Value: 25, Color: "#FFA500"},   // Chaud - orange
			{Value: 30, Color: "#FF4500"},   // Très chaud - rouge-orange
			{Value: 35, Color: "#FF0000"},   // Chaud intense - rouge
			{Value: 45, Color: "#8B0000"},   // Chaleur extrême - rouge foncé
			{Value: 60, Color: "#4B0000"},   // Chaleur dangereuse - rouge très foncé
		},
	},
	"wind_speed": {
		Name:        "wind_speed",
		Icon:        "wind.svg",
		Unit:        "m/s",
		ShowPalette: true,
		Colors: []ColorPoint{
			{Value: 0, Color: "rgba(255,255,255,0)"},      // Calm - transparent
			{Value: 1, Color: "rgba(230,247,255,0.1)"},    // Light air - très peu visible
			{Value: 2, Color: "rgba(179,229,255,0.3)"},    // Light breeze - légèrement visible
			{Value: 3, Color: "rgba(128,212,255,0.5)"},    // Gentle breeze - semi-transparent
			{Value: 5, Color: "rgba(77,195,255,0.7)"},     // Moderate breeze - plus visible
			{Value: 7, Color: "#1AB2FF"},                  // Fresh breeze - bleu vif opaque
			{Value: 10, Color: "#00A0E6"},                 // Strong breeze - bleu intense
			{Value: 12, Color: "#0080B3"},                 // Near gale - bleu foncé
			{Value: 15, Color: "#66CC66"},                 // Moderate gale - vert (comme Windy)
			{Value: 18, Color: "#99DD00"},                 // Fresh gale - vert clair
			{Value: 20, Color: "#FFCC00"},                 // Strong gale - jaune
			{Value: 25, Color: "#FF9900"},                 // Storm - orange
			{Value: 30, Color: "#FF6600"},                 // Violent storm - rouge-orange
			{Value: 35, Color: "#FF3300"},                 // Hurricane - rouge vif
			{Value: 40, Color: "#CC0000"},                 // Hurricane force - rouge foncé
			{Value: 50, Color: "#990066"},                 // Extreme - violet
		},
	},
	"rainfall_accumulation": {
		Name:        "rainfall",
		Icon:        "rainfall.svg",
		Unit:        "mm",
		ShowPalette: true,
		Colors: []ColorPoint{
			{Value: 0, Color: "rgba(255,255,255, 0)"},
			{Value: 0.01, Color: "rgba(255,255,255, 0.01)"},
			{Value: 0.02, Color: "rgba(255,255,255, 0.02)"},
			{Value: 0.03, Color: "rgba(255,255,255, 0.03)"},
			{Value: 0.04, Color: "rgba(255,255,255, 0.04)"},
			{Value: 0.05, Color: "rgba(255,255,255, 0.05)"},
			{Value: 0.06, Color: "rgba(255,255,255, 0.06)"},
			{Value: 0.07, Color: "rgba(255,255,255, 0.07)"},
			{Value: 0.08, Color: "rgba(255,255,255, 0.08)"},
			{Value: 0.09, Color: "rgba(255,255,255, 0.09)"},
			{Value: 0.1, Color: "rgba(255,255,255, 0.2)"},
			{Value: 0.11, Color: "rgba(255,255,255, 0.23)"},
			{Value: 0.12, Color: "rgba(255,255,255, 0.26)"},
			{Value: 0.13, Color: "rgba(255,255,255, 0.29)"},
			{Value: 0.14, Color: "rgba(255,255,255, 0.32)"},
			{Value: 0.15, Color: "rgba(255,255,255, 0.35)"},
			{Value: 0.16, Color: "rgba(255,255,255, 0.38)"},
			{Value: 0.17, Color: "rgba(255,255,255, 0.41)"},
			{Value: 0.18, Color: "rgba(255,255,255, 0.44)"},
			{Value: 0.19, Color: "rgba(255,255,255, 0.47)"},
			{Value: 0.2, Color: "rgba(255,255,255, 0.50)"},
			{Value: 0.21, Color: "rgba(255,255,255, 0.53)"},
			{Value: 0.22, Color: "rgba(255,255,255, 0.56)"},
			{Value: 0.23, Color: "rgba(255,255,255, 0.59)"},
			{Value: 0.24, Color: "rgba(255,255,255, 0.62)"},
			{Value: 0.25, Color: "rgba(255,255,255, 0.65)"},
			{Value: 0.26, Color: "rgba(255,255,255, 0.68)"},
			{Value: 0.27, Color: "rgba(255,255,255, 0.71)"},
			{Value: 0.35, Color: "#e1f2fc"},
			{Value: 0.5, Color: "#5fd4f4"},
			{Value: 0.75, Color: "#45c2f0"},
			{Value: 1, Color: "#35c2f0"},
			{Value: 1.5, Color: "#25b2ec"},
			{Value: 2, Color: "#1aa7ec"},
			{Value: 3.5, Color: "#28c9c6"},
			{Value: 5, Color: "#37eba5"},
			{Value: 7.5, Color: "#42dc86"},
			{Value: 10, Color: "#4cd167"},
			{Value: 15, Color: "#64c855"},
			{Value: 20, Color: "#7bc043"},
			{Value: 25, Color: "#8ed545"},
			{Value: 30, Color: "#a0eb4c"},
			{Value: 35, Color: "#b0e34a"},
			{Value: 40, Color: "#c0d647"},
			{Value: 45, Color: "#e0dc48"},
			{Value: 50, Color: "#ffe04a"},
			{Value: 55, Color: "#ffd04b"},
			{Value: 60, Color: "#ffc04c"},
			{Value: 65, Color: "#ffaf40"},
			{Value: 70, Color: "#ff9e3d"},
			{Value: 75, Color: "#ff8f3f"},
			{Value: 80, Color: "#ff7f41"},
			{Value: 85, Color: "#ff7250"},
			{Value: 90, Color: "#ff6a5a"},
			{Value: 95, Color: "#ed6b67"},
			{Value: 100, Color: "#e56b6f"},
			{Value: 125, Color: "#ea4755"},
			{Value: 150, Color: "#ef233c"},
			{Value: 175, Color: "#e41333"},
			{Value: 200, Color: "#d90429"},
			{Value: 250, Color: "#ae012e"},
			{Value: 300, Color: "#8d0033"},
			{Value: 350, Color: "#8c003c"},
			{Value: 400, Color: "#8c0045"},
			{Value: 450, Color: "#89023f"},
			{Value: 500, Color: "#85023e"},
			{Value: 550, Color: "#7e0145"},
			{Value: 600, Color: "#77004d"},
			{Value: 650, Color: "#700057"},
			{Value: 700, Color: "#6a0061"},
			{Value: 750, Color: "#5e0062"},
			{Value: 800, Color: "#560063"},
			{Value: 850, Color: "#4d0071"},
			{Value: 900, Color: "#440080"},
		},
	},
	"temperature": {
		Name:        "temperature",
		Icon:        "temperature.svg",
		Unit:        "°C",
		ShowPalette: true,
		Colors: []ColorPoint{
			{Value: -40, Color: "#000080"},  // Très froid - bleu foncé
			{Value: -20, Color: "#0000FF"},  // Froid extrême - bleu
			{Value: -10, Color: "#4169E1"},  // Froid - bleu royal
			{Value: 0, Color: "#ADD8E6"},    // Frais - bleu clair
			{Value: 10, Color: "#90EE90"},   // Doux - vert clair
			{Value: 15, Color: "#FFFFE0"},   // Neutre - jaune très clair
			{Value: 20, Color: "#FFD700"},   // Agréable - jaune doré
			{Value: 25, Color: "#FFA500"},   // Chaud - orange
			{Value: 30, Color: "#FF4500"},   // Très chaud - rouge-orange
			{Value: 35, Color: "#FF0000"},   // Chaud intense - rouge
			{Value: 45, Color: "#8B0000"},   // Chaleur extrême - rouge foncé
			{Value: 60, Color: "#4B0000"},   // Chaleur dangereuse - rouge très foncé
		},
	},
	"humidity": {
		Name:        "humidity",
		Icon:        "humidity.svg",
		Unit:        "%",
		ShowPalette: true,
		Colors: []ColorPoint{
			{Value: 0, Color: "#8B4513"},
			{Value: 20, Color: "#9EA913"},
			{Value: 30, Color: "#A8DB13"},
			{Value: 40, Color: "#9DFF22"},
			{Value: 50, Color: "#61FF54"},
			{Value: 60, Color: "#25FF86"},
			{Value: 70, Color: "#00FFB8"},
			{Value: 80, Color: "#00CDEA"},
			{Value: 90, Color: "#009BFF"},
			{Value: 99, Color: "#006EFF"},
		},
	},
	"cloud_cover": {
		Name:        "cloud_cover",
		Icon:        "cloud_cover.svg",
		Unit:        "%",
		ShowPalette: false,
		Colors:      []ColorPoint{
			{Value: 0, Color: "rgba(255,255,255, 0.01)"},
			{Value: 1, Color: "rgba(255,255,255, 0.01)"},
			{Value: 5, Color: "rgba(255,255,255, 0.02)"},
			{Value: 10, Color: "rgba(255,255,255, 0.05)"},
			{Value: 20, Color: "rgba(255,255,255, 0.10)"},
			{Value: 30, Color: "rgba(255,255,255, 0.17)"},
			{Value: 40, Color: "rgba(255,255,255, 0.25)"},
			{Value: 50, Color: "rgba(255,255,255, 0.33)"},
			{Value: 60, Color: "rgba(255,255,255, 0.41)"},
			{Value: 70, Color: "rgba(255,255,255, 0.5)"},
			{Value: 80, Color: "rgba(255,255,255, 0.58)"},
			{Value: 90, Color: "rgba(255,255,255, 0.66)"},
			{Value: 100, Color: "rgba(255,255,255, 0.95)"},
		},
	},
}

func parseHexColor(s string) (color.RGBA, error) {
	var r, g, b, a uint8
	a = 255
	var err error
	if len(s) == 4 { // #RGB
		_, err = fmt.Sscanf(s, "#%1x%1x%1x", &r, &g, &b)
		if err == nil {
			r |= r << 4
			g |= g << 4
			b |= b << 4
		}
	} else if len(s) == 7 { // #RRGGBB
		_, err = fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	} else if len(s) == 9 { // #RRGGBBAA
		_, err = fmt.Sscanf(s, "#%02x%02x%02x%02x", &r, &g, &b, &a)
	} else {
		err = fmt.Errorf("invalid hex color format")
	}

	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid hex color format: %s", s)
	}
	return color.RGBA{R: r, G: g, B: b, A: a}, nil
}

func parseRGBAColor(s string) (color.RGBA, error) {
	trimmed := strings.TrimSpace(s)
	if !strings.HasPrefix(trimmed, "rgba(") || !strings.HasSuffix(trimmed, ")") {
		return color.RGBA{}, fmt.Errorf("invalid rgba format: %s", s)
	}
	content := trimmed[5 : len(trimmed)-1]
	parts := strings.Split(content, ",")
	if len(parts) != 4 {
		return color.RGBA{}, fmt.Errorf("invalid rgba format: expected 4 parts in %s", s)
	}

	r, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid red value in rgba: %s", parts[0])
	}
	g, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid green value in rgba: %s", parts[1])
	}
	b, err := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid blue value in rgba: %s", parts[2])
	}
	a, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid alpha value in rgba: %s", parts[3])
	}

	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a * 255)}, nil
}

func parseColor(s string) (color.RGBA, error) {
	if strings.HasPrefix(s, "#") {
		return parseHexColor(s)
	}
	if strings.HasPrefix(s, "rgba") {
		return parseRGBAColor(s)
	}
	return color.RGBA{}, fmt.Errorf("unsupported color format: %s", s)
}

func interpolateColor(c1, c2 color.RGBA, factor float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c1.R) + factor*(float64(c2.R)-float64(c1.R))),
		G: uint8(float64(c1.G) + factor*(float64(c2.G)-float64(c1.G))),
		B: uint8(float64(c1.B) + factor*(float64(c2.B)-float64(c1.B))),
		A: uint8(float64(c1.A) + factor*(float64(c2.A)-float64(c1.A))),
	}
}

func GetColorForValue(layer string, value float64) string {
	if layer == "cloud_cover" {
		alpha := value / 100.0
		if alpha < 0 {
			alpha = 0
		}
		if alpha > 1 {
			alpha = 1
		}
		return fmt.Sprintf("rgba(200, 200, 200, %.2f)", alpha)
	}

	palette, ok := palettes[layer]
	if !ok {
		return "rgba(0,0,0,0)" // Default transparent
	}

	colors := palette.Colors
	if len(colors) == 0 {
		return "rgba(0,0,0,0)"
	}

	sort.Slice(colors, func(i, j int) bool {
		return colors[i].Value < colors[j].Value
	})

	if value <= colors[0].Value {
		return colors[0].Color
	}
	if value >= colors[len(colors)-1].Value {
		return colors[len(colors)-1].Color
	}

	for i := 0; i < len(colors)-1; i++ {
		p1 := colors[i]
		p2 := colors[i+1]
		if value >= p1.Value && value <= p2.Value {
			c1, err1 := parseColor(p1.Color)
			c2, err2 := parseColor(p2.Color)

			if err1 != nil || err2 != nil {
				return "rgba(0,0,0,0)" // Error parsing color
			}

			factor := (value - p1.Value) / (p2.Value - p1.Value)
			interpColor := interpolateColor(c1, c2, factor)

			if interpColor.A == 255 {
				return fmt.Sprintf("#%02x%02x%02x", interpColor.R, interpColor.G, interpColor.B)
			}
			return fmt.Sprintf("rgba(%d, %d, %d, %.2f)", interpColor.R, interpColor.G, interpColor.B, float64(interpColor.A)/255.0)
		}
	}

	return "rgba(0,0,0,0)"
}

func GetPaletteAsJSON() string {
	fullPalettes := make(map[string]Palette)

	for key, p := range palettes {
		if key == "cloud_cover" {
			cloudPalette := p
			cloudPalette.Colors = make([]ColorPoint, 101)
			for i := 0; i <= 100; i++ {
				alpha := float64(i) / 100.0
				cloudPalette.Colors[i] = ColorPoint{
					Value: float64(i),
					Color: fmt.Sprintf("rgba(200, 200, 200, %.2f)", alpha),
				}
			}
			fullPalettes[key] = cloudPalette
		} else {
			fullPalettes[key] = p
		}
	}

	b, err := json.Marshal(fullPalettes)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func GetValueFromColor(layer string, color string) float64 {
	palette, ok := palettes[layer]
	if !ok {
		return 0
	}

	colors := palette.Colors
	if len(colors) == 0 {
		return 0
	}

	// Sort colors by value to ensure correct interpolation
	sort.Slice(colors, func(i, j int) bool {
		return colors[i].Value < colors[j].Value
	})

	// Parse the input color
	inputColor, err := parseColor(color)
	if err != nil {
		return 0
	}

	// Check for exact matches first
	for _, colorPoint := range colors {
		paletteColor, err := parseColor(colorPoint.Color)
		if err != nil {
			continue
		}
		if inputColor.R == paletteColor.R && inputColor.G == paletteColor.G && 
		   inputColor.B == paletteColor.B && inputColor.A == paletteColor.A {
			return colorPoint.Value
		}
	}

	// Find the best interpolation match
	bestInterpolation := -1
	bestFactor := 0.0
	minInterpolationError := math.MaxFloat64
	
	for i := 0; i < len(colors)-1; i++ {
		p1 := colors[i]
		p2 := colors[i+1]
		
		c1, err1 := parseColor(p1.Color)
		c2, err2 := parseColor(p2.Color)
		
		if err1 != nil || err2 != nil {
			continue
		}
		
		// Calculate the best interpolation factor for this color pair
		factor := calculateBestInterpolationFactor(inputColor, c1, c2)
		
		// Generate the interpolated color at this factor
		interpolatedColor := interpolateColor(c1, c2, factor)
		
		// Calculate how close this interpolated color is to our input color
		error := colorDistance(inputColor, interpolatedColor)
		
		// If this is the best match so far, remember it
		if error < minInterpolationError {
			minInterpolationError = error
			bestInterpolation = i
			bestFactor = factor
		}
	}
	
	// If we found a good interpolation match, return the interpolated value
	if bestInterpolation >= 0 && minInterpolationError < 50 { // Reasonable threshold
		p1 := colors[bestInterpolation]
		p2 := colors[bestInterpolation+1]
		
		// Clamp factor to [0, 1] range for extrapolation control
		if bestFactor < 0 {
			bestFactor = 0
		}
		if bestFactor > 1 {
			bestFactor = 1
		}
		
		return p1.Value + bestFactor*(p2.Value-p1.Value)
	}
	
	// Fallback to closest single color match
	minDistance := math.MaxFloat64
	bestValue := 0.0
	
	for _, colorPoint := range colors {
		paletteColor, err := parseColor(colorPoint.Color)
		if err != nil {
			continue
		}
		
		distance := colorDistance(inputColor, paletteColor)
		if distance < minDistance {
			minDistance = distance
			bestValue = colorPoint.Value
		}
	}
	
	return bestValue
}

// calculateBestInterpolationFactor calculates the best interpolation factor to match a target color
func calculateBestInterpolationFactor(targetColor, c1, c2 color.RGBA) float64 {
	// Calculate interpolation factors for each color component
	factors := make([]float64, 0, 4)
	
	// Red component
	if c2.R != c1.R {
		factor := float64(targetColor.R-c1.R) / float64(c2.R-c1.R)
		factors = append(factors, factor)
	}
	
	// Green component
	if c2.G != c1.G {
		factor := float64(targetColor.G-c1.G) / float64(c2.G-c1.G)
		factors = append(factors, factor)
	}
	
	// Blue component
	if c2.B != c1.B {
		factor := float64(targetColor.B-c1.B) / float64(c2.B-c1.B)
		factors = append(factors, factor)
	}
	
	// Alpha component
	if c2.A != c1.A {
		factor := float64(targetColor.A-c1.A) / float64(c2.A-c1.A)
		factors = append(factors, factor)
	}
	
	// If no components differ, return 0 (colors are identical)
	if len(factors) == 0 {
		return 0.0
	}
	
	// Calculate weighted average, giving more weight to components with larger differences
	totalWeight := 0.0
	weightedSum := 0.0
	
	weights := []float64{
		math.Abs(float64(c2.R - c1.R)),
		math.Abs(float64(c2.G - c1.G)),
		math.Abs(float64(c2.B - c1.B)),
		math.Abs(float64(c2.A - c1.A)),
	}
	
	factorIndex := 0
	for _, weight := range weights {
		if weight > 0 {
			if factorIndex < len(factors) {
				weightedSum += factors[factorIndex] * weight
				totalWeight += weight
				factorIndex++
			}
		}
	}
	
	if totalWeight == 0 {
		return 0.0
	}
	
	return weightedSum / totalWeight
}

// isColorBetween checks if a color could be an interpolation between two colors
func isColorBetween(color, c1, c2 color.RGBA) bool {
	// Check if color components are within the range of c1 and c2
	rInRange := (color.R >= min(c1.R, c2.R) && color.R <= max(c1.R, c2.R)) ||
		(c1.R == c2.R && color.R == c1.R)
	gInRange := (color.G >= min(c1.G, c2.G) && color.G <= max(c1.G, c2.G)) ||
		(c1.G == c2.G && color.G == c1.G)
	bInRange := (color.B >= min(c1.B, c2.B) && color.B <= max(c1.B, c2.B)) ||
		(c1.B == c2.B && color.B == c1.B)
	aInRange := (color.A >= min(c1.A, c2.A) && color.A <= max(c1.A, c2.A)) ||
		(c1.A == c2.A && color.A == c1.A)
	
	return rInRange && gInRange && bInRange && aInRange
}

// getInterpolationFactor calculates the interpolation factor for a color between two colors
func getInterpolationFactor(color, c1, c2 color.RGBA) float64 {
	// Use the component with the largest difference to calculate the factor
	maxDiff := 0.0
	factor := 0.0
	
	if diff := math.Abs(float64(c2.R) - float64(c1.R)); diff > maxDiff {
		maxDiff = diff
		if diff > 0 {
			factor = (float64(color.R) - float64(c1.R)) / diff
		}
	}
	if diff := math.Abs(float64(c2.G) - float64(c1.G)); diff > maxDiff {
		maxDiff = diff
		if diff > 0 {
			factor = (float64(color.G) - float64(c1.G)) / diff
		}
	}
	if diff := math.Abs(float64(c2.B) - float64(c1.B)); diff > maxDiff {
		maxDiff = diff
		if diff > 0 {
			factor = (float64(color.B) - float64(c1.B)) / diff
		}
	}
	if diff := math.Abs(float64(c2.A) - float64(c1.A)); diff > maxDiff {
		maxDiff = diff
		if diff > 0 {
			factor = (float64(color.A) - float64(c1.A)) / diff
		}
	}
	
	return factor
}

// colorDistance calculates the Euclidean distance between two colors
func colorDistance(c1, c2 color.RGBA) float64 {
	dr := float64(c1.R) - float64(c2.R)
	dg := float64(c1.G) - float64(c2.G)
	db := float64(c1.B) - float64(c2.B)
	da := float64(c1.A) - float64(c2.A)
	
	return math.Sqrt(dr*dr + dg*dg + db*db + da*da)
}

// min returns the minimum of two uint8 values
func min(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two uint8 values
func max(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}