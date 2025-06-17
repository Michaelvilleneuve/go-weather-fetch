package storage

import (
	"bytes"
	"encoding/json"
	"compress/gzip"
	"path/filepath"
	"fmt"
	"os"
	"io"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
)


func Save(data [][]float64, hour string, original_time string) (string, error) {
	payload := map[string]interface{}{
		"data": data,
		"hour": hour,
		"original_time": original_time,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write(jsonPayload)
	gz.Close()

	os.WriteFile(fmt.Sprintf("tmp/rainfall_%s.json.gz", hour), buf.Bytes(), 0644)

	return "", nil
}

func IsUpToDate(dt string) bool {
	lastDownloaded, err := os.ReadFile("storage/current_run_datetime.txt")
	isUpToDate := bytes.Equal(lastDownloaded, []byte(dt))

	if err != nil || !isUpToDate {
		os.WriteFile("tmp/current_run_datetime.txt", []byte(dt), 0644)
		return false
	}

	return isUpToDate
}

func RollOut() {
	files, err := filepath.Glob("tmp/rainfall_*.json.gz")
	if err != nil {
		utils.Log("Error during globbing: " + err.Error())
		return
	}

	for _, src := range files {
		dst := filepath.Join("storage", filepath.Base(src))
		err := moveFile(src, dst)
		if err != nil {
			utils.Log("Error moving file " + src + ": " + err.Error())
		}
	}

	// Move the current_run_datetime.txt file
	err = moveFile("tmp/current_run_datetime.txt", "storage/current_run_datetime.txt")
	if err != nil {
		utils.Log("Error moving current_run_datetime.txt: " + err.Error())
	}
}

func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	err = dstFile.Sync()
	if err != nil {
		return err
	}

	return os.Remove(src)
}