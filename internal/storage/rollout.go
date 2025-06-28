package storage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/arome"
	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
)

func WatchForWorkerRollout() {
	go func() {
		for {
			files, err := filepath.Glob("tmp/*.mbtiles")
			if err != nil {
				utils.Log("Error during globbing: " + err.Error())
				continue
			}

			totalHours, err := strconv.Atoi(os.Getenv("TOTAL_HOURS"))
			if err != nil {
				utils.Log("Error converting TOTAL_HOURS to int: " + err.Error())
				continue
			}

			expectedNumberOfMBTilesFiles := totalHours * len(arome.Configuration().GetLayerNames())

			if (len(files) < expectedNumberOfMBTilesFiles) {
				utils.Log(fmt.Sprintf("Not enough files to roll out update, waiting for more files (%d/%d)", len(files), expectedNumberOfMBTilesFiles))
				time.Sleep(5 * time.Second)
				continue
			}

			RollOut(arome.Configuration().Packages)
		}
	}()
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
	targetHost := os.Getenv("ROLLOUT_TARGET_HOST")
	utils.Log("Rolling out remotely to " + targetHost)
	
	for _, forecastPackage := range forecastPackages {
		for _, commonName := range forecastPackage.GetLayerNames() {
			files, err := filepath.Glob(fmt.Sprintf("tmp/%s_*.geojson.mbtiles", commonName))
			if err != nil {
				utils.Log("Error during globbing: " + err.Error())
				return
			}

			for _, src := range files {
				dst := filepath.Join("/data/storage/weather-fetch/tmp", filepath.Base(src))
				cmd := exec.Command("scp", src, fmt.Sprintf("%s:%s", targetHost, dst))
				output, err := cmd.CombinedOutput()
				if err != nil {
					utils.Log(fmt.Sprintf("Error copying file %s: %s\nOutput: %s", src, err.Error(), string(output)))
				}
			}
		}

		err := moveFile(fmt.Sprintf("tmp/%s_current_run_datetime.txt", forecastPackage.Name), fmt.Sprintf("storage/%s_current_run_datetime.txt", forecastPackage.Name))
		if err != nil {
			utils.Log("Error moving file " + fmt.Sprintf("tmp/%s_current_run_datetime.txt", forecastPackage.Name) + ": " + err.Error())
		}

		src := fmt.Sprintf("storage/%s_current_run_datetime.txt", forecastPackage.Name)
		dst := fmt.Sprintf("/data/storage/weather-fetch/tmp/%s_current_run_datetime.txt", forecastPackage.Name)
		cmd := exec.Command("scp", src, fmt.Sprintf("%s:%s", targetHost, dst))

		output, err := cmd.CombinedOutput()

		if err != nil {
			utils.Log(fmt.Sprintf("Error copying file %s: %s\nOutput: %s", src, err.Error(), string(output)))
		}

		CleanUpFiles(forecastPackage.Name)
		utils.Log("Done.")
	}
}