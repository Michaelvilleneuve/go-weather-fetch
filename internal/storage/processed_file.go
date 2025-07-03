package storage

type ProcessedFile struct {
	Model string
	Format string
	Run string
	Layer string
	Hour string
}

func (processedFile ProcessedFile) GetFileName() string {
	return processedFile.Model + "_" + processedFile.Run + "_" + processedFile.Layer + "_" + processedFile.Hour
}

func (processedFile ProcessedFile) GetFinalTmpFilePath() string {
	if processedFile.Format == "cog" {
		return processedFile.GetTmpCOGFilePath()
	} 
	return processedFile.GetTmpMBTilesFilePath()
}

func (processedFile ProcessedFile) GetFinalRolledOutFilePath() string {
	if processedFile.Format == "cog" {
		return processedFile.GetRolledOutCOGFileName()
	}
	return processedFile.GetRolledOutMBTilesFileName()
}

func (processedFile ProcessedFile) GetTmpGeoJSONFilePath() string {
	return "tmp/" + processedFile.GetFileName() + ".geojson"
}

func (processedFile ProcessedFile) GetTmpMBTilesFilePath() string {
	return "tmp/" + processedFile.GetFileName() + ".mbtiles"
}

func (processedFile ProcessedFile) GetRolledOutMBTilesFileName() string {
	return "storage/" + processedFile.GetFileName() + ".mbtiles"
}

func (processedFile ProcessedFile) GetTmpASCIIGridFilePath() string {
	return "tmp/" + processedFile.GetFileName() + ".asc"
}

func (processedFile ProcessedFile) GetTmpGeoTIFFFilePath() string {
	return "tmp/" + processedFile.GetFileName() + ".tif"
}

func (processedFile ProcessedFile) GetTmpCOGFilePath() string {
	return "tmp/" + processedFile.GetFileName() + "_cog.tif"
}

func (processedFile ProcessedFile) GetRolledOutCOGFileName() string {
	return "storage/" + processedFile.GetFileName() + "_cog.tif"
}

func (processedFile ProcessedFile) GetTmpRGBFilePath(band string) string {
	return "tmp/" + processedFile.GetFileName() + "_" + band + ".dat"
}

func (processedFile ProcessedFile) GetTmpVRTFilePath() string {
	return "tmp/" + processedFile.GetFileName() + ".vrt"
}

func (processedFile ProcessedFile) GetTmpRGBGeoTIFFPath(band string) string {
	return "tmp/" + processedFile.GetFileName() + "_" + band + ".tif"
}
