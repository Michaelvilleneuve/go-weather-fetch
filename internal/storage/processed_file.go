package storage

type ProcessedFile struct {
	Model string
	Run string
	Layer string
	Hour string
}

func (processedFile ProcessedFile) GetFileName() string {
	return processedFile.Model + "_" + processedFile.Run + "_" + processedFile.Layer + "_" + processedFile.Hour
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
