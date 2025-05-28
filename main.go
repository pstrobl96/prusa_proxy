package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/alecthomas/kingpin"
	"github.com/gorilla/mux"
)

var (
	configFile    = kingpin.Flag("config", "Configuration file for prusa_proxy.").Default("./prusa.yml").ExistingFile()
	listenPort    = kingpin.Flag("port", "Address where to expose port for gathering metrics.").Default("31100").String()
	configuration Config
)

func getJobID(url string, username string, password string) (int, error) {
	status, err := getStatus(url, username, password)

	if err != nil {
		return 0, err
	}

	return int(status.Job.ID), nil
}

func homepageHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`<html>
    <head><title>prusa_proxy</title></head>
    <body>
    <h1>prusa_proxy</h1>
	<h3>Implemented Endpoints</h3>
	<ul>
		<li>/pause</li>
		<li>/resume</li>
	</ul> 
	</body>
    </html>`))
}

func getNecessities(ip string) (string, string, string, error) {
	username := getUsername(ip, configuration.Printers)
	if username == "" {
		return "", "", "", fmt.Errorf("username not found for the printer with IP: %s", ip)
	}
	password := getPassword(ip, configuration.Printers)
	if password == "" {
		return "", "", "", fmt.Errorf("password not found for the printer with IP: %s", ip)
	}
	jobID, err := getJobID(ip, username, password)
	if jobID == 0 {
		return "", "", "", fmt.Errorf("no job found for the printer with IP: %s", ip)
	}
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get job ID for the printer with IP: %s, error: %v", ip, err)
	}
	return username, password, strconv.Itoa(jobID), nil
}

func getOperationURL(ip string, jobID string, operation string) (string, error) {
	return "http://" + ip + "/api/v1/job/" + jobID + "/" + operation, nil
}

func pausePrinterHandler(w http.ResponseWriter, r *http.Request) {
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	username, password, jobID, err := getNecessities(req.IP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	url, err := getOperationURL(req.IP, jobID, "pause")
	if err != nil {
		http.Error(w, "Failed to get operation URL: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := putDigestRequest(url, username, password)

	if err != nil {
		http.Error(w, "Failed to pause the printer: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		http.Error(w, "Failed to pause the printer: "+resp.Status, resp.StatusCode)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func resumePrinterHandler(w http.ResponseWriter, r *http.Request) {
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	username, password, jobID, err := getNecessities(req.IP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	url, err := getOperationURL(req.IP, jobID, "resume")
	if err != nil {
		http.Error(w, "Failed to get operation URL: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := putDigestRequest(url, username, password)

	if err != nil {
		http.Error(w, "Failed to pause the printer: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		http.Error(w, "Failed to pause the printer: "+resp.Status, resp.StatusCode)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func stopPrinterHandler(w http.ResponseWriter, r *http.Request) {
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	username, password, jobID, err := getNecessities(req.IP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	url, err := getOperationURL(req.IP, jobID, "")
	if err != nil {
		http.Error(w, "Failed to get operation URL: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := deleteDigestRequest(url, username, password)

	if err != nil {
		http.Error(w, "Failed to pause the printer: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		http.Error(w, "Failed to pause the printer: "+resp.Status, resp.StatusCode)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func pauseAllPrintersHandler(w http.ResponseWriter, r *http.Request) {
	for _, printer := range configuration.Printers {
		username, password, jobID, err := getNecessities(printer.Address)
		if err != nil {
			w.Write([]byte("Error getting configuration for printer " + printer.Address + ": " + err.Error() + "\n"))
			continue
		}
		url, err := getOperationURL(printer.Address, jobID, "pause")
		if err != nil {
			w.Write([]byte("Failed to get operation URL for printer " + printer.Address + ": " + err.Error() + "\n"))
			continue
		}
		resp, err := putDigestRequest(url, username, password)
		if err != nil {
			w.Write([]byte("Failed to pause the printer: " + err.Error() + "\n"))
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			w.Write([]byte("Failed to pause the printer: " + resp.Status + "\n"))
			continue
		}
		w.Write([]byte("Printer " + printer.Address + " paused successfully.\n"))
	}
}

func resumeAllPrintersHandler(w http.ResponseWriter, r *http.Request) {
	for _, printer := range configuration.Printers {
		username, password, jobID, err := getNecessities(printer.Address)
		if err != nil {
			w.Write([]byte("Error getting configuration for printer " + printer.Address + ": " + err.Error() + "\n"))
			continue
		}
		url, err := getOperationURL(printer.Address, jobID, "resume")
		if err != nil {
			w.Write([]byte("Failed to get operation URL for printer " + printer.Address + ": " + err.Error() + "\n"))
			continue
		}
		resp, err := putDigestRequest(url, username, password)
		if err != nil {
			w.Write([]byte("Failed to resume the printer: " + err.Error() + "\n"))
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			w.Write([]byte("Failed to resume the printer: " + resp.Status + "\n"))
			continue
		}
		w.Write([]byte("Printer " + printer.Address + " resumed successfully.\n"))
	}
}

func stopAllPrintersHandler(w http.ResponseWriter, r *http.Request) {
	for _, printer := range configuration.Printers {
		username, password, jobID, err := getNecessities(printer.Address)
		if err != nil {
			w.Write([]byte("Error getting configuration for printer " + printer.Address + ": " + err.Error() + "\n"))
			continue
		}
		url, err := getOperationURL(printer.Address, jobID, "")
		if err != nil {
			w.Write([]byte("Failed to get operation URL for printer " + printer.Address + ": " + err.Error() + "\n"))
			continue
		}
		resp, err := deleteDigestRequest(url, username, password)
		if err != nil {
			w.Write([]byte("Failed to stop the printer: " + err.Error() + "\n"))
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			w.Write([]byte("Failed to stop the printer: " + resp.Status + "\n"))
			continue
		}
		w.Write([]byte("Printer " + printer.Address + " stopped successfully.\n"))
	}

}

func main() {
	kingpin.Parse()
	log.Println("Starting prusa_proxy with configuration file:", *configFile)
	log.Println("Listening on port: localhost:" + *listenPort)
	if _, err := os.Stat(*configFile); os.IsNotExist(err) {
		log.Panic("Configuration file does not exist: " + *configFile)
	}

	var err error

	configuration, err = LoadConfig(*configFile)

	if err != nil {
		log.Panic("Error loading configuration file " + err.Error())
	}

	router := mux.NewRouter()

	router.HandleFunc("/", homepageHandler).Methods("GET")
	router.HandleFunc("/pause", pausePrinterHandler).Methods("POST")
	router.HandleFunc("/resume", resumePrinterHandler).Methods("POST")
	router.HandleFunc("/stop", stopPrinterHandler).Methods("POST")
	router.HandleFunc("/all/pause", pauseAllPrintersHandler).Methods("POST")
	router.HandleFunc("/all/resume", resumeAllPrintersHandler).Methods("POST")
	router.HandleFunc("/all/stop", stopAllPrintersHandler).Methods("POST")
	log.Fatal(http.ListenAndServe(":"+*listenPort, router))
}
