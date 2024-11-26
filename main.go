package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// NOTE Variabili globali
var serverPort = "8080" // porta in ascolto sul server
var dataFolder = "data" // nome della cartella contenete i file json con le configurazioni dei singoli endpoint

type EndpointConfig struct {
	Request struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	} `json:"request"`
	Response struct {
		Body interface{} `json:"body"`
	} `json:"response"`
}

func loadData(configDir string, mux *http.ServeMux) error {
	// Legge i file JSON dalla cartella
	files, err := os.ReadDir(configDir)
	if err != nil {
		return fmt.Errorf("impossibile leggere la directory %s: %w", configDir, err)
	}
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			configPath := filepath.Join(configDir, file.Name())
			// Legge il contenuto del file JSON
			data, err := os.ReadFile(configPath)
			if err != nil {
				return fmt.Errorf("errore nella lettura del file %s: %w", configPath, err)
			}
			// Parse JSON nel modello
			var endpointConfig EndpointConfig
			err = json.Unmarshal(data, &endpointConfig)
			if err != nil {
				return fmt.Errorf("errore nel parsing del JSON %s: %w", configPath, err)
			}
			// Registra gli endpoint nell'handler
			createHandler(mux, endpointConfig)
		}
	}
	return nil
}

func createHandler(mux *http.ServeMux, config EndpointConfig) {
	// Aggiunge rotta dinamica
	mux.HandleFunc(config.Request.Path, func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Method: %s\t Path: %s\n", r.Method, r.URL)
		// Verifica il metodo
		if r.Method != config.Request.Method {
			http.Error(w, "Method Not Allowed / Metodo non consentito", http.StatusMethodNotAllowed)
			return
		}
		// Imposta il content type
		w.Header().Set("Content-Type", "application/json")
		// Risponde con il body configurato nel file JSON
		responseData, err := json.Marshal(config.Response.Body)
		if err != nil {
			http.Error(w, "Internal Server Error / Errore interno del server", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(responseData)
	})
}

func main() {
	// Configurazione del server con mutex
	mux := http.NewServeMux()
	// Leggi e registra tutti gli endpoint dai file JSON
	err := loadData(dataFolder, mux)
	if err != nil {
		log.Fatalf("Errore durante il caricamento delle route: %v", err)
	}
	// Avvio del server
	fmt.Printf("Avvio server in ascolto sulla porta %s\n", serverPort)
	log.Fatal(http.ListenAndServe(":"+serverPort, mux))
}
