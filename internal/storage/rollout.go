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

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/arome"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
)

func ReceiveRollout() {
	http.HandleFunc("/rollout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("X-Rollout-Secret") != os.Getenv("ROLLOUT_SECRET") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		// Read the mbtiles file from the request body
		file, header, err := r.FormFile("mbtiles")
		if err != nil {
			utils.Log("Error receiving mbtiles file: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Create the destination file in storage directory
		dst, err := os.Create(filepath.Join("storage", header.Filename))
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

		rollOutIfRunIsComplete()

		w.Write([]byte("Mbtiles file received and saved"))
	})
}

func rollOutIfRunIsComplete() {
	files, err := filepath.Glob("tmp/*.mbtiles")
	if err != nil {
		utils.Log("Error during globbing: " + err.Error())
		return
	}

	totalHours, err := strconv.Atoi(os.Getenv("TOTAL_HOURS"))
	if err != nil {
		utils.Log("Error converting TOTAL_HOURS to int: " + err.Error())
		return
	}

	expectedNumberOfMBTilesFiles := totalHours * len(arome.Configuration().GetLayerNames())

	if (len(files) < expectedNumberOfMBTilesFiles) {
		utils.Log(fmt.Sprintf("Not enough files to roll out update, waiting for more files (%d/%d)", len(files), expectedNumberOfMBTilesFiles))
		return
	}

	RollOut(arome.Configuration().Packages)
}

func RollOut(forecastPackages []arome.AromePackage) {
	if os.Getenv("ROLLOUT_TARGET_HOST") == "" {
		rolloutLocally(forecastPackages)
	} else {
		rolloutRemotely(forecastPackages)
	}
}

func rolloutLocally(forecastPackages []arome.AromePackage) {
	utils.Log("Rolling out locally")
	
	for _, forecastPackage := range forecastPackages {
		for _, commonName := range forecastPackage.GetLayerNames() {
			files, err := filepath.Glob(fmt.Sprintf("tmp/%s_*.geojson.mbtiles", commonName))
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

		err := moveFile(fmt.Sprintf("tmp/%s_current_run_datetime.txt", forecastPackage.Name), fmt.Sprintf("storage/%s_current_run_datetime.txt", forecastPackage.Name))
		if err != nil {
			utils.Log("Error moving file " + fmt.Sprintf("tmp/%s_current_run_datetime.txt", forecastPackage.Name) + ": " + err.Error())
		}

		CleanUpFiles(forecastPackage.Name)
	}
}

func rolloutRemotely(forecastPackages []arome.AromePackage) {
	utils.Log("Rolling out remotely")
	
	for _, forecastPackage := range forecastPackages {
		for _, commonName := range forecastPackage.GetLayerNames() {
			files, err := filepath.Glob(fmt.Sprintf("tmp/%s_*.geojson.mbtiles", commonName))
			if err != nil {
				utils.Log("Error during globbing: " + err.Error())
				return
			}

			for _, src := range files {
				err := uploadFileToHost(src)
				if err != nil {
					utils.Log(fmt.Sprintf("Error uploading file %s: %s", src, err.Error()))
				}
			}
		}

		err := moveFile(fmt.Sprintf("tmp/%s_current_run_datetime.txt", forecastPackage.Name), fmt.Sprintf("storage/%s_current_run_datetime.txt", forecastPackage.Name))
		if err != nil {
			utils.Log("Error moving file " + fmt.Sprintf("tmp/%s_current_run_datetime.txt", forecastPackage.Name) + ": " + err.Error())
		}

		src := fmt.Sprintf("storage/%s_current_run_datetime.txt", forecastPackage.Name)
		err = uploadFileToHost(src)
		if err != nil {
			utils.Log(fmt.Sprintf("Error uploading file %s: %s", src, err.Error()))
		}

		CleanUpFiles(forecastPackage.Name)
		utils.Log("Done.")
	}
}

func uploadFileToHost(filePath string) error {
	targetHost := os.Getenv("ROLLOUT_TARGET_HOST")
	rolloutSecret := os.Getenv("ROLLOUT_SECRET")

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("mbtiles", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/rollout", targetHost), &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Rollout-Secret", rolloutSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	utils.Log(fmt.Sprintf("Successfully uploaded %s", filepath.Base(filePath)))
	return nil
}