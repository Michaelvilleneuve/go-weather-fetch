package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"net/http"
	"time"
	"github.com/joho/godotenv"
)


func SendToApi(data [][]float64, hour string, original_time string) (string, error) {
	apiUrl := os.Getenv("LVDLV_API_URL") + "/rainfall_data"
	headers := map[string]string{
		"Content-Type": "application/json",
		"Authorization": os.Getenv("LVDLV_API_KEY"),
	}

	payload := map[string]interface{}{
		"data": data,
		"hour": hour,
		"original_time": original_time,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	client := &http.Client{}

	for i := 0; i < 3; i++ {
		req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonPayload))
		if err != nil {
			return "", err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", headers["Authorization"])

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}

		if resp.StatusCode == 201 {
			return "", nil
		}

		if i < 2 {
			time.Sleep(2 * time.Second)
		}
	}

	return "", fmt.Errorf("failed to get successful response after 3 attempts")
}