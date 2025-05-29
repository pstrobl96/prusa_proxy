package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/icholy/digest"
)

type request struct {
	IP string `json:"ip"`
}

// Job is a struct that contains data about print job
type Job struct {
	State string `json:"state"`
	Job   struct {
		EstimatedPrintTime float64 `json:"estimatedPrintTime"`
		File               struct {
			Name    string  `json:"name"`
			Path    string  `json:"path"`
			Display string  `json:"display"`
			Size    float64 `json:"size"`
			Origin  string  `json:"origin"`
			Date    float64 `json:"date"`
		} `json:"file"`
		AveragePrintTime any    `json:"averagePrintTime"`
		LastPrintTime    any    `json:"lastPrintTime"`
		Filament         any    `json:"filament"`
		User             string `json:"user"`
	} `json:"job"`
	Progress struct {
		PrintTimeLeft       float64 `json:"printTimeLeft"`
		Completion          float64 `json:"completion"`
		PrintTime           float64 `json:"printTime"`
		Filepos             float64 `json:"filepos"`
		PrintTimeLeftOrigin string  `json:"printTimeLeftOrigin"`
		PosZMm              float64 `json:"pos_z_mm"`
		PrintSpeed          float64 `json:"printSpeed"`
		FlowFactor          float64 `json:"flow_factor"`
	} `json:"progress"`
}

// GetJob is used to get the printer's job API endpoint
func getState(url string, username string, password string) (string, error) {
	var job Job
	response, err := getDigestRequest("http://"+url+"/api/job", username, password)

	if err != nil {
		return "", err
	}

	err = json.Unmarshal(response, &job)

	return job.State, err
}

// GetStatus is used to get Buddy status endpoint
func getStatus(url string, username string, password string) (status, error) {
	var status status
	response, err := getDigestRequest("http://"+url+"/api/v1/status", username, password)

	if err != nil {
		return status, err
	}

	err = json.Unmarshal(response, &status)

	return status, err
}

func putDigestRequest(url string, username string, password string) (*http.Response, error) {
	client := &http.Client{
		Transport: &digest.Transport{
			Username: username,
			Password: password,
		},
	}

	putBody := []byte(`{}`)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(putBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil { /*  */
		log.Fatal("Error sending PUT request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading response body: " + err.Error())
	}

	log.Printf("Response Status: %s\n", resp.Status)
	log.Printf("Response Body: %s\n", body)

	if resp.StatusCode > 200 && resp.StatusCode < 300 {
		log.Printf("PUT request with Digest Authentication successful!")
	} else {
		log.Printf("PUT request with Digest Authentication failed or unexpected status code.")
	}
	client.CloseIdleConnections()

	return resp, err
}

func getDigestRequest(url string, username string, password string) ([]byte, error) {
	var (
		res    *http.Response
		result []byte
		err    error
	)

	client := &http.Client{
		Transport: &digest.Transport{
			Username: username,
			Password: password,
		},
	}

	res, err = client.Get(url)

	if err != nil {
		return result, err
	}
	result, err = io.ReadAll(res.Body)
	res.Body.Close()

	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return nil, err
	}

	return result, nil
}

func deleteDigestRequest(url string, username string, password string) (*http.Response, error) {
	client := &http.Client{
		Transport: &digest.Transport{
			Username: username,
			Password: password,
		},
	}

	req, err := http.NewRequest(http.MethodDelete, url, nil)

	if err != nil {
		log.Fatalf("Error creating DELETE request: %v", err)
	}

	resp, err := client.Do(req)

	if err != nil {
		log.Fatalf("Error sending DELETE request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	log.Printf("Response Status for DELETE: %s\n", resp.Status)
	log.Printf("Response Body for DELETE: %s\n", body)

	// DELETE requests typically return 200 OK or 204 No Content for success.
	// You might adjust this success check based on your API's expected behavior.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("DELETE request with Digest Authentication successful!")
	} else {
		log.Printf("DELETE request with Digest Authentication failed or unexpected status code.")
	}
	client.CloseIdleConnections()

	return resp, err
}

type status struct {
	Job struct {
		ID            float64 `json:"id"`
		Progress      float64 `json:"progress"`
		TimeRemaining float64 `json:"time_remaining"`
		TimePrinting  float64 `json:"time_printing"`
	} `json:"job"`
	Printer struct {
		State        string  `json:"state"`
		TempBed      float64 `json:"temp_bed"`
		TargetBed    float64 `json:"target_bed"`
		TempNozzle   float64 `json:"temp_nozzle"`
		TargetNozzle float64 `json:"target_nozzle"`
		AxisX        float64 `json:"axis_x"`
		AxisY        float64 `json:"axis_y"`
		AxisZ        float64 `json:"axis_z"`
		Flow         float64 `json:"flow"`
		Speed        float64 `json:"speed"`
		FanHotend    float64 `json:"fan_hotend"`
		FanPrint     float64 `json:"fan_print"`
	} `json:"printer"`
}
