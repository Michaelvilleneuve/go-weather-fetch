package storage

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
)

func saveAsRaster(data [][]float64, processedFile ProcessedFile) (ProcessedFile, error) {
	utils.Log("Saving raster for " + processedFile.GetFileName())

	processedFile.Format = "cog"

	// Create colorized GeoTIFF file directly
	err := createColorizedGeoTIFF(data, processedFile)
	if err != nil {
		utils.Log("Error creating colorized GeoTIFF: " + err.Error())
		return processedFile, err
	}

	// Convert GeoTIFF to COG
	err = convertToCOG(processedFile)
	if err != nil {
		utils.Log("Error converting to COG: " + err.Error())
		return processedFile, err
	}

	// Clean up intermediate files
	os.Remove(processedFile.GetTmpGeoTIFFFilePath())

	utils.Log("Successfully created colorized COG file: " + processedFile.GetTmpCOGFilePath())

	return processedFile, nil
}

func createColorizedGeoTIFF(data [][]float64, processedFile ProcessedFile) error {
	if len(data) == 0 {
		return fmt.Errorf("no data provided")
	}

	// Calculate grid bounds and resolution
	bounds := calculateBounds(data)
	gridInfo := calculateGridInfo(data, bounds)

	// Create RGBA grids with colorization (including alpha channel)
	rGrid, gGrid, bGrid, aGrid, valueGrid := createColorizedRasterGrids(data, gridInfo, bounds, processedFile.Layer)

	// Create separate GeoTIFF files for each RGBA band
	err := createSingleBandGeoTIFF(rGrid, processedFile.GetTmpRGBGeoTIFFPath("r"), gridInfo, bounds)
	if err != nil {
		return fmt.Errorf("error creating red band GeoTIFF: %v", err)
	}

	err = createSingleBandGeoTIFF(gGrid, processedFile.GetTmpRGBGeoTIFFPath("g"), gridInfo, bounds)
	if err != nil {
		return fmt.Errorf("error creating green band GeoTIFF: %v", err)
	}

	err = createSingleBandGeoTIFF(bGrid, processedFile.GetTmpRGBGeoTIFFPath("b"), gridInfo, bounds)
	if err != nil {
		return fmt.Errorf("error creating blue band GeoTIFF: %v", err)
	}

	err = createSingleBandGeoTIFF(aGrid, processedFile.GetTmpRGBGeoTIFFPath("a"), gridInfo, bounds)
	if err != nil {
		return fmt.Errorf("error creating alpha band GeoTIFF: %v", err)
	}

	err = createSingleBandGeoTIFF(valueGrid, processedFile.GetTmpRGBGeoTIFFPath("value"), gridInfo, bounds)
	if err != nil {
		return fmt.Errorf("error creating value band GeoTIFF: %v", err)
	}

	// Create VRT file to combine RGBA bands
	err = createVRTFile(processedFile, gridInfo, bounds)
	if err != nil {
		return err
	}

	// Convert VRT to final GeoTIFF with all bands (RGBA + Value)
	cmd := exec.Command("gdal_translate",
		"-of", "GTiff",
		"-co", "COMPRESS=LZW",
		"-co", "TILED=YES",
		"-co", "BLOCKXSIZE=64",
		"-co", "BLOCKYSIZE=64",
		// Remove PHOTOMETRIC=RGB to allow extra bands
		// "-co", "PHOTOMETRIC=RGB",
		"-co", "ALPHA=YES", // Enable alpha channel
		processedFile.GetTmpVRTFilePath(),
		processedFile.GetTmpGeoTIFFFilePath(),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gdal_translate failed: %s\nOutput: %s", err, string(output))
	}

	// Clean up temporary files (including the value band file now that VRT is processed)
	os.Remove(processedFile.GetTmpVRTFilePath())
	os.Remove(processedFile.GetTmpRGBGeoTIFFPath("r"))
	os.Remove(processedFile.GetTmpRGBGeoTIFFPath("g"))
	os.Remove(processedFile.GetTmpRGBGeoTIFFPath("b"))
	os.Remove(processedFile.GetTmpRGBGeoTIFFPath("a"))
	os.Remove(processedFile.GetTmpRGBGeoTIFFPath("value"))
	return nil
}

func createSingleBandGeoTIFF(grid []float32, filepath string, gridInfo GridInfo, bounds Bounds) error {
	// Create ASCII grid file first
	asciiPath := filepath + ".asc"
	file, err := os.Create(asciiPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write ASCII grid header
	fmt.Fprintf(file, "ncols         %d\n", gridInfo.Width)
	fmt.Fprintf(file, "nrows         %d\n", gridInfo.Height)
	fmt.Fprintf(file, "xllcorner     %.6f\n", bounds.MinLon)
	fmt.Fprintf(file, "yllcorner     %.6f\n", bounds.MinLat)
	fmt.Fprintf(file, "cellsize      %.6f\n", gridInfo.PixelSizeX)
	fmt.Fprintf(file, "NODATA_value  -1\n")

	// Determine if this is a value band based on filename
	isValueBand := strings.Contains(filepath, "_value.tif")
	
	// Write grid data
	for row := 0; row < gridInfo.Height; row++ {
		var rowValues []string
		for col := 0; col < gridInfo.Width; col++ {
			index := row*gridInfo.Width + col
			value := grid[index]
			if isValueBand {
				// Use higher precision for value bands
				rowValues = append(rowValues, fmt.Sprintf("%.3f", value))
			} else {
				// Use integer precision for RGBA bands
				rowValues = append(rowValues, fmt.Sprintf("%.0f", value))
			}
		}
		fmt.Fprintf(file, "%s\n", strings.Join(rowValues, " "))
	}
	file.Close()

	// Determine data type for gdal_translate
	var outputType string
	if isValueBand {
		outputType = "Float32"
	} else {
		outputType = "Byte"
	}

	// Convert ASCII to GeoTIFF
	cmd := exec.Command("gdal_translate",
		"-of", "GTiff",
		"-co", "COMPRESS=LZW",
		"-a_srs", "EPSG:4326",
		"-ot", outputType,
		asciiPath,
		filepath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gdal_translate failed for %s: %s\nOutput: %s", filepath, err, string(output))
	}

	// Clean up ASCII file
	os.Remove(asciiPath)
	return nil
}

func createVRTFile(processedFile ProcessedFile, gridInfo GridInfo, bounds Bounds) error {
	vrtFile, err := os.Create(processedFile.GetTmpVRTFilePath())
	if err != nil {
		return err
	}
	defer vrtFile.Close()

	rFileName := processedFile.GetFileName() + "_r.tif"
	gFileName := processedFile.GetFileName() + "_g.tif"
	bFileName := processedFile.GetFileName() + "_b.tif"
	aFileName := processedFile.GetFileName() + "_a.tif"
	valueFileName := processedFile.GetFileName() + "_value.tif"

	vrtContent := fmt.Sprintf(`<VRTDataset rasterXSize="%d" rasterYSize="%d">
  <SRS>EPSG:4326</SRS>
  <GeoTransform>%.6f,%.6f,0.0,%.6f,0.0,%.6f</GeoTransform>
  <VRTRasterBand dataType="Byte" band="1">
    <ColorInterp>Red</ColorInterp>
    <SimpleSource>
      <SourceFilename relativeToVRT="1">%s</SourceFilename>
      <SourceBand>1</SourceBand>
      <SourceProperties RasterXSize="%d" RasterYSize="%d" DataType="Byte"/>
      <SrcRect xOff="0" yOff="0" xSize="%d" ySize="%d"/>
      <DstRect xOff="0" yOff="0" xSize="%d" ySize="%d"/>
    </SimpleSource>
  </VRTRasterBand>
  <VRTRasterBand dataType="Byte" band="2">
    <ColorInterp>Green</ColorInterp>
    <SimpleSource>
      <SourceFilename relativeToVRT="1">%s</SourceFilename>
      <SourceBand>1</SourceBand>
      <SourceProperties RasterXSize="%d" RasterYSize="%d" DataType="Byte"/>
      <SrcRect xOff="0" yOff="0" xSize="%d" ySize="%d"/>
      <DstRect xOff="0" yOff="0" xSize="%d" ySize="%d"/>
    </SimpleSource>
  </VRTRasterBand>
  <VRTRasterBand dataType="Byte" band="3">
    <ColorInterp>Blue</ColorInterp>
    <SimpleSource>
      <SourceFilename relativeToVRT="1">%s</SourceFilename>
      <SourceBand>1</SourceBand>
      <SourceProperties RasterXSize="%d" RasterYSize="%d" DataType="Byte"/>
      <SrcRect xOff="0" yOff="0" xSize="%d" ySize="%d"/>
      <DstRect xOff="0" yOff="0" xSize="%d" ySize="%d"/>
    </SimpleSource>
  </VRTRasterBand>
  <VRTRasterBand dataType="Byte" band="4">
    <ColorInterp>Alpha</ColorInterp>
    <SimpleSource>
      <SourceFilename relativeToVRT="1">%s</SourceFilename>
      <SourceBand>1</SourceBand>
      <SourceProperties RasterXSize="%d" RasterYSize="%d" DataType="Byte"/>
      <SrcRect xOff="0" yOff="0" xSize="%d" ySize="%d"/>
      <DstRect xOff="0" yOff="0" xSize="%d" ySize="%d"/>
    </SimpleSource>
  </VRTRasterBand>
  <VRTRasterBand dataType="Float32" band="5">
    <Description>Value</Description>
    <SimpleSource>
      <SourceFilename relativeToVRT="1">%s</SourceFilename>
      <SourceBand>1</SourceBand>
      <SourceProperties RasterXSize="%d" RasterYSize="%d" DataType="Float32"/>
      <SrcRect xOff="0" yOff="0" xSize="%d" ySize="%d"/>
      <DstRect xOff="0" yOff="0" xSize="%d" ySize="%d"/>
    </SimpleSource>
  </VRTRasterBand>
</VRTDataset>`,
		gridInfo.Width, gridInfo.Height,
		bounds.MinLon, gridInfo.PixelSizeX, bounds.MaxLat, -gridInfo.PixelSizeY,
		// Red band
		rFileName,
		gridInfo.Width, gridInfo.Height,
		gridInfo.Width, gridInfo.Height,
		gridInfo.Width, gridInfo.Height,
		// Green band
		gFileName,
		gridInfo.Width, gridInfo.Height,
		gridInfo.Width, gridInfo.Height,
		gridInfo.Width, gridInfo.Height,
		// Blue band
		bFileName,
		gridInfo.Width, gridInfo.Height,
		gridInfo.Width, gridInfo.Height,
		gridInfo.Width, gridInfo.Height,
		// Alpha band
		aFileName,
		gridInfo.Width, gridInfo.Height,
		gridInfo.Width, gridInfo.Height,
		gridInfo.Width, gridInfo.Height,
		// Value band
		valueFileName,
		gridInfo.Width, gridInfo.Height,
		gridInfo.Width, gridInfo.Height,
		gridInfo.Width, gridInfo.Height,
	)

	_, err = vrtFile.WriteString(vrtContent)
	return err
}

func convertToCOG(processedFile ProcessedFile) error {
	utils.Log("Converting GeoTIFF to COG for " + processedFile.GetFileName())

	cmd := exec.Command("gdal_translate",
		"-of", "GTiff",
		"-co", "TILED=YES",
		"-co", "BLOCKXSIZE=128", 
		"-co", "BLOCKYSIZE=128",
		"-co", "COMPRESS=LZW",
		"-co", "COPY_SRC_OVERVIEWS=YES",
		"-co", "ALPHA=YES", // Preserve alpha channel
		processedFile.GetTmpGeoTIFFFilePath(),
		processedFile.GetTmpCOGFilePath(),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gdal_translate failed: %s\nOutput: %s", err, string(output))
	}

	// Add overviews for better performance
	cmd = exec.Command("gdaladdo",
		"-r", "average",
		processedFile.GetTmpCOGFilePath(),
		"2", "4", "8", "16",
	)

	output, err = cmd.CombinedOutput()
	if err != nil {
		utils.Log("Warning: Failed to add overviews: " + err.Error())
		// Don't return error as overviews are optional
	}

	return nil
}

type Bounds struct {
	MinLon, MaxLon, MinLat, MaxLat float64
}

type GridInfo struct {
	Width, Height           int
	PixelSizeX, PixelSizeY float64
}

func calculateBounds(data [][]float64) Bounds {
	if len(data) == 0 {
		return Bounds{}
	}

	bounds := Bounds{
		MinLon: data[0][0],
		MaxLon: data[0][0],
		MinLat: data[0][1],
		MaxLat: data[0][1],
	}

	for _, point := range data {
		lon, lat := point[0], point[1]
		if lon < bounds.MinLon {
			bounds.MinLon = lon
		}
		if lon > bounds.MaxLon {
			bounds.MaxLon = lon
		}
		if lat < bounds.MinLat {
			bounds.MinLat = lat
		}
		if lat > bounds.MaxLat {
			bounds.MaxLat = lat
		}
	}

	return bounds
}

func calculateGridInfo(data [][]float64, bounds Bounds) GridInfo {
	// Extract unique coordinates to determine grid resolution
	lonSet := make(map[float64]bool)
	latSet := make(map[float64]bool)

	for _, point := range data {
		lonSet[point[0]] = true
		latSet[point[1]] = true
	}

	// Convert to sorted slices
	var lons, lats []float64
	for lon := range lonSet {
		lons = append(lons, lon)
	}
	for lat := range latSet {
		lats = append(lats, lat)
	}

	sort.Float64s(lons)
	sort.Float64s(lats)

	// Calculate pixel size from the smallest difference between consecutive coordinates
	pixelSizeX := 0.01 // Default fallback
	pixelSizeY := 0.01 // Default fallback

	if len(lons) > 1 {
		minDiff := math.Abs(lons[1] - lons[0])
		for i := 1; i < len(lons)-1; i++ {
			diff := math.Abs(lons[i+1] - lons[i])
			if diff > 0 && diff < minDiff {
				minDiff = diff
			}
		}
		pixelSizeX = minDiff
	}

	if len(lats) > 1 {
		minDiff := math.Abs(lats[1] - lats[0])
		for i := 1; i < len(lats)-1; i++ {
			diff := math.Abs(lats[i+1] - lats[i])
			if diff > 0 && diff < minDiff {
				minDiff = diff
			}
		}
		pixelSizeY = minDiff
	}

	// Calculate grid dimensions based on bounds and pixel size
	width := int(math.Ceil((bounds.MaxLon-bounds.MinLon)/pixelSizeX)) + 1
	height := int(math.Ceil((bounds.MaxLat-bounds.MinLat)/pixelSizeY)) + 1

	return GridInfo{
		Width:      width,
		Height:     height,
		PixelSizeX: pixelSizeX,
		PixelSizeY: pixelSizeY,
	}
}

func createColorizedRasterGrids(data [][]float64, gridInfo GridInfo, bounds Bounds, layer string) ([]float32, []float32, []float32, []float32, []float32) {
	// Initialize grids with transparent values
	rGrid := make([]float32, gridInfo.Width*gridInfo.Height)
	gGrid := make([]float32, gridInfo.Width*gridInfo.Height)
	bGrid := make([]float32, gridInfo.Width*gridInfo.Height)
	aGrid := make([]float32, gridInfo.Width*gridInfo.Height)
	valueGrid := make([]float32, gridInfo.Width*gridInfo.Height)

	// Initialize with transparent background
	for i := range rGrid {
		rGrid[i] = 0
		gGrid[i] = 0
		bGrid[i] = 0
		aGrid[i] = 0
		valueGrid[i] = 0
	}

	// Populate grids with colorized data points
	for _, point := range data {
		lon, lat, value := point[0], point[1], point[2]

		// Get color from palette
		colorStr := GetColorForValue(layer, value)
		rgba, err := parseColorString(colorStr)
		if err != nil {
			continue // Skip invalid colors
		}

		// Check if color should be rendered as transparent
		if shouldBeTransparent(rgba) {
			continue // Skip transparent colors, leaving them transparent
		}

		// Calculate grid indices
		col := int((lon - bounds.MinLon) / gridInfo.PixelSizeX)
		row := int((bounds.MaxLat - lat) / gridInfo.PixelSizeY) // Note: row calculation for north-up orientation

		// Ensure indices are within bounds
		if col >= 0 && col < gridInfo.Width && row >= 0 && row < gridInfo.Height {
			index := row*gridInfo.Width + col
			rGrid[index] = float32(rgba.R)
			gGrid[index] = float32(rgba.G)
			bGrid[index] = float32(rgba.B)
			aGrid[index] = float32(rgba.A)
			valueGrid[index] = float32(value)
		}
	}

	return rGrid, gGrid, bGrid, aGrid, valueGrid
}

// shouldBeTransparent determines if a color should be rendered as transparent
// based on its alpha value or luminance/brightness
func shouldBeTransparent(rgba color.RGBA) bool {
	// Check if color has significant alpha transparency
	// Consider colors with alpha < 128 (50% opacity) as transparent
	if rgba.A < 128 {
		return true
	}

	// Also check if color is dark (for opaque dark colors)
	return isDarkColor(rgba)
}

// isDarkColor determines if a color is dark based on its luminance/brightness
func isDarkColor(rgba color.RGBA) bool {
	// Calculate luminance using the standard formula: 0.299*R + 0.587*G + 0.114*B
	luminance := 0.299*float64(rgba.R) + 0.587*float64(rgba.G) + 0.114*float64(rgba.B)
	
	// Consider colors with luminance below 50 (out of 255) as "dark"
	// This threshold can be adjusted based on requirements
	darkThreshold := 50.0
	
	return luminance < darkThreshold
}

func parseColorString(colorStr string) (color.RGBA, error) {
	return parseColor(colorStr)
}