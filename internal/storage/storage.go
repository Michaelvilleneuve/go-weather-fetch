package storage

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
)


func AnticipateExit() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		utils.Log("Cleaning up before exit...")
		CleanUpFiles("")
		os.Exit(0)
	}()
}


func CleanUpFiles(pattern string) {
	files, err := os.ReadDir("./tmp")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if len(pattern) > 0 && !strings.Contains(file.Name(), pattern) {
			continue
		}
		os.Remove("./tmp/" + file.Name())
	}
}


func Save(data [][]float64, packageName string, hour string, original_time string) (string, error) {
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

	os.WriteFile(fmt.Sprintf("tmp/%s_%s.json.gz", packageName, hour), buf.Bytes(), 0644)

	return "", nil
}

func IsUpToDate(packageName string, dt string) bool {
	lastDownloaded, err := os.ReadFile(fmt.Sprintf("storage/%s_current_run_datetime.txt", packageName))
	isUpToDate := bytes.Equal(lastDownloaded, []byte(dt))

	if err != nil || !isUpToDate {
		os.WriteFile(fmt.Sprintf("tmp/%s_current_run_datetime.txt", packageName), []byte(dt), 0644)
		return false
	}

	return isUpToDate
}

func RollOut(packageName string, commonNames []string) {
	for _, commonName := range commonNames {
		files, err := filepath.Glob(fmt.Sprintf("tmp/%s_*.json.gz", commonName))
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

	}

	
	// Move the current_run_datetime.txt file
	err := moveFile(fmt.Sprintf("tmp/%s_current_run_datetime.txt", packageName), fmt.Sprintf("storage/%s_current_run_datetime.txt", packageName))
	if err != nil {
		utils.Log("Error moving file " + fmt.Sprintf("tmp/%s_current_run_datetime.txt", packageName) + ": " + err.Error())
	}

	CleanUpFiles(packageName)
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