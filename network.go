package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// UploadFileToMemory reads a file from an HTTP request and returns its content as a byte slice.
func UploadFileToMemory(r *http.Request) ([]byte, string, error) {
	// ParseMultipartForm parses a multipart form, including file uploads.
	// The argument is the maximum amount of memory to use for parsing the form data.
	// Files larger than this will be stored on disk.
	err := r.ParseMultipartForm(10 << 20) // 10 MB limit for form data in memory
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse multipart form: %w", err)
	}

	// Get the file from the request. "file" is the name of the input field in the HTML form.
	file, handler, err := r.FormFile("file")
	if err != nil {
		return nil, "", fmt.Errorf("failed to get file from form: %w", err)
	}
	defer file.Close() // Ensure the file is closed after processing

	// Read the file content into a byte slice.
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file content: %w", err)
	}

	return fileBytes, handler.Filename, nil
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	fileContent, filename, err := UploadFileToMemory(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error uploading file: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Example: Upload the file to the printer via Digest Auth
	printerURL := "http://192.168.20.29/api/v1/files/usb//" + filename // Replace <PRINTER_IP> as needed
	username := "maker"                                                // Replace with actual username
	password := "ozhLHCHFf9aoRr6"                                      // Replace with actual password

	client := &http.Client{
		Transport: &digest.Transport{
			Username: username,
			Password: password,
		},
	}

	req, err := http.NewRequest(http.MethodPut, printerURL, bytes.NewReader(fileContent))
	if err != nil {
		http.Error(w, "Failed to create upload request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/gcode+binary")
	req.Header.Set("Overwrite", "?1")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to upload file to printer: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		http.Error(w, "Printer upload failed: "+string(body), resp.StatusCode)
		return
	}

	w.Write([]byte("File uploaded to printer successfully!"))

}

func uploadPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Upload File</title>
		</head>
		<body>
			<h1>Upload File</h1>
			<form id="uploadForm" action="/upload" method="post" enctype="multipart/form-data" target="hidden_iframe">
				<div id="file">
		<input type="file" id="fileInput" name="file" accept=".gcode, .bgcode" required></div>
				<input type="submit" value="Upload">
			</form>
			<iframe name="hidden_iframe" style="display:none;"></iframe>
			<div id="result"></div>
			<script>
				document.getElementById('uploadForm').onsubmit = function() {
					var fileInput = document.getElementById('fileInput');
					var file = fileInput.files[0];
					if (!file) {
						document.getElementById('result').innerText = "No file selected.";
						return false;
					}
					var size = file.size;
					var kbps = 200 * 1024; // assume 200kbps
					var seconds = Math.ceil(size / kbps);
					document.getElementById('result').innerText = "Uploading... (Estimated time: " + seconds + "s)";
					setTimeout(function() {
						document.getElementById('result').innerText = "Upload to Grafana complete (check printer for result).";
					}, seconds * 1000);
					return true;
				};
			</script>
		</body>
		</html>
	`)
}
