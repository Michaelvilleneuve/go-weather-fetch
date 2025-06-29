package storage

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/model"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
)

func ReceiveRollout() {
	http.HandleFunc("/rollout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Authorization") != os.Getenv("ROLLOUT_SECRET") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		processedFile := ProcessedFile{
			Model: r.FormValue("model"),
			Run: r.FormValue("run"), 
			Layer: r.FormValue("layer"),
			Hour: r.FormValue("hour"),
		}

		// Read the mbtiles file from the request body
		file, _, err := r.FormFile("mbtiles")
		if err != nil {
			utils.Log("Error receiving mbtiles file: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()

		dst, err := os.Create(processedFile.GetRolledOutMBTilesFileName())
		if err != nil {
			utils.Log("Error creating destination file: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// Copy the uploaded file to storage
		_, err = io.Copy(dst, file)
		if err != nil {
			utils.Log("Error saving mbtiles file: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		rollOutIfRunIsComplete(processedFile)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Mbtiles file received and saved"))
	})
}

func rollOutIfRunIsComplete(processedFile ProcessedFile) {
	files, err := filepath.Glob(fmt.Sprintf("tmp/%s_%s_*.mbtiles", processedFile.Model, processedFile.Run))
	if err != nil {
		utils.Log("Error during globbing: " + err.Error())
		return
	}

	totalHours, err := strconv.Atoi(os.Getenv("TOTAL_HOURS"))
	if err != nil {
		utils.Log("Error converting TOTAL_HOURS to int: " + err.Error())
		return
	}

	modelConfiguration := model.GetModel(processedFile.Model)
	allLayersFromModel := modelConfiguration.GetLayerNames()

	expectedNumberOfMBTilesFiles := totalHours * len(allLayersFromModel)

	if (len(files) < expectedNumberOfMBTilesFiles) {
		utils.Log(fmt.Sprintf("Not enough files to roll out update, waiting for more files (%d/%d)", len(files), expectedNumberOfMBTilesFiles))
		return
	}

	rolloutLocally(processedFile)
}

func rolloutLocally(processedFile ProcessedFile) {
	utils.Log("Rolling out locally")
	
	for _, commonName := range model.GetModel(processedFile.Model).GetLayerNames() {
		files, err := filepath.Glob(fmt.Sprintf("tmp/%s_*.mbtiles", commonName))
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
}

func RolloutRemotely(processedFile ProcessedFile) {
	targetHost := os.Getenv("ROLLOUT_TARGET_HOST")
	rolloutSecret := os.Getenv("ROLLOUT_SECRET")

	file, err := os.Open(processedFile.GetTmpMBTilesFilePath())
	if err != nil {
		utils.Log("Error opening file: " + err.Error())
		return
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	writer.WriteField("model", processedFile.Model)
	writer.WriteField("run", processedFile.Run)
	writer.WriteField("layer", processedFile.Layer)
	writer.WriteField("hour", processedFile.Hour)

	part, err := writer.CreateFormFile("mbtiles", filepath.Base(processedFile.GetTmpMBTilesFilePath()))
	if err != nil {
		utils.Log("Error creating form file: " + err.Error())
		return
	}

	_, err = io.Copy(part, file)
	if err != nil {
		utils.Log("Error copying file content: " + err.Error())
		return
	}

	err = writer.Close()
	if err != nil {
		utils.Log("Error closing multipart writer: " + err.Error())
		return
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/rollout", targetHost), &buf)
	if err != nil {
		utils.Log("Error creating request: " + err.Error())
		return
	}

	req.Header.Set("Authorization", rolloutSecret)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		utils.Log("Error sending request: " + err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		utils.Log("Server returned status " + strconv.Itoa(resp.StatusCode))
		return
	}

	utils.Log(fmt.Sprintf("Successfully uploaded %s", processedFile.GetTmpMBTilesFilePath()))
}