package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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

type printerOperation func(url, username, password string) (*http.Response, error)

func handlePrinterOperation(w http.ResponseWriter, r *http.Request, operation string, method printerOperation, opName string) {
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

	url, err := getOperationURL(req.IP, jobID, operation)
	if err != nil {
		http.Error(w, "Failed to get operation URL: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := method(url, username, password)
	if err != nil {
		http.Error(w, "Failed to "+opName+" the printer: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		http.Error(w, "Failed to "+opName+" the printer: "+resp.Status, resp.StatusCode)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func pausePrinterHandler(w http.ResponseWriter, r *http.Request) {
	handlePrinterOperation(w, r, "pause", putDigestRequest, "pause")
}

func resumePrinterHandler(w http.ResponseWriter, r *http.Request) {
	handlePrinterOperation(w, r, "resume", putDigestRequest, "resume")
}

func stopPrinterHandler(w http.ResponseWriter, r *http.Request) {
	handlePrinterOperation(w, r, "", deleteDigestRequest, "stop")
}

func handleAllPrintersOperation(w http.ResponseWriter, r *http.Request, operation string, method printerOperation, opName string) {
	for _, printer := range configuration.Printers {
		username, password, jobID, err := getNecessities(printer.Address)
		if err != nil {
			w.Write([]byte("Error getting configuration for printer " + printer.Address + ": " + err.Error() + "\n"))
			continue
		}
		url, err := getOperationURL(printer.Address, jobID, operation)
		if err != nil {
			w.Write([]byte("Failed to get operation URL for printer " + printer.Address + ": " + err.Error() + "\n"))
			continue
		}
		resp, err := method(url, username, password)
		if err != nil {
			w.Write([]byte("Failed to " + opName + " the printer: " + err.Error() + "\n"))
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			w.Write([]byte("Failed to " + opName + " the printer: " + resp.Status + "\n"))
			continue
		}
		w.Write([]byte("Printer " + printer.Address + " " + opName + "d successfully.\n"))
	}
}

func pauseAllPrintersHandler(w http.ResponseWriter, r *http.Request) {
	handleAllPrintersOperation(w, r, "pause", putDigestRequest, "pause")
}

func resumeAllPrintersHandler(w http.ResponseWriter, r *http.Request) {
	handleAllPrintersOperation(w, r, "resume", putDigestRequest, "resume")
}

func stopAllPrintersHandler(w http.ResponseWriter, r *http.Request) {
	// Reverse order for stop operation
	for i := len(configuration.Printers) - 1; i >= 0; i-- {
		printer := configuration.Printers[i]
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

		state, err := getStatus(printer.Address, username, password)
		if err != nil {
			w.Write([]byte("Error getting status for printer " + printer.Address + ": " + err.Error() + "\n"))
			continue
		}
		if state.Printer.State == "Stopping" {
			w.Write([]byte("Printer " + printer.Address + " is stopping, waiting for it to finish.\n"))
			time.Sleep(1 * time.Second)
			i = i - 1
		}

		getJobID(printer.Address, username, password) // Ensure the job is cleared

		if jobID != "" {
			w.Write([]byte(printer.Address + " - still not stopped, trying again.\n"))
			i = i - 1
		} else {
			w.Write([]byte("No job found for printer " + printer.Address + ".\n"))
		}

		w.Write([]byte("Printer " + printer.Address + " stopped successfully.\n"))
	}
}

func exportState(w http.ResponseWriter, r *http.Request) {

	w.Write([]byte("\n# TYPE prusa_proxy_printer_state gauge\n"))
	for _, printer := range configuration.Printers {
		username := getUsername(printer.Address, configuration.Printers)
		if username == "" {
			log.Printf("Username not found for printer %s", printer.Address)
			continue
		}
		password := getPassword(printer.Address, configuration.Printers)
		if password == "" {
			log.Printf("Password not found for printer %s", printer.Address)
			continue
		}
		state, err := getState(printer.Address, username, password)
		if err != nil {
			log.Printf("Error getting status for printer %s: %v", printer.Address, err)
			continue
		}
		w.Write(fmt.Appendf(nil, "prusa_proxy_printer_state{printer=\"%s\", state=\"%s\"} %d\n",
			printer.Address, state, 1))
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
	router.HandleFunc("/metrics", exportState).Methods("GET")
	log.Fatal(http.ListenAndServe(":"+*listenPort, router))
}
