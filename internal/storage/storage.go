package storage

import (
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/utils"
)

func Save(data [][]float64, processedFile ProcessedFile) (ProcessedFile, error) {
	return saveAsRaster(data, processedFile)
}

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