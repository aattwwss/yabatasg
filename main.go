package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"encoding/json"
	"math/rand"
)

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFiles embed.FS

func main() {
	// Configure structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create HTTP handler for static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		slog.Error("Failed to create static filesystem", "error", err)
		os.Exit(1)
	}
	http.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Define route for root path
	http.HandleFunc("GET /", homeHandler)
	http.HandleFunc("GET /api/v1/busArrival", corsMiddleware(arrivalHandler))

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	slog.Info("Starting server", "port", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		slog.Warn("404 Not Found", "path", r.URL.Path, "method", r.Method)
		http.NotFound(w, r)
		return
	}

	// Parse template from embedded FS
	tmpl, err := template.ParseFS(templateFiles, "templates/index.html")
	if err != nil {
		slog.Error("Template parsing failed", "error", err, "template", "index.html")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err = tmpl.Execute(w, nil); err != nil {
		slog.Error("Template execution failed", "error", err, "template", "index.html")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	slog.Info("Served template",
		"template", "index.html",
		"path", r.URL.Path,
		"method", r.Method,
		"client_ip", r.RemoteAddr,
	)
}

func arrivalHandler(w http.ResponseWriter, r *http.Request) {
	// Set response header to indicate JSON content
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	busStopCode := query.Get("BusStopCode")
	serviceNo := query.Get("ServiceNo")

	// You can use the parameters for logging, validation, or business logic
	// For example:
	if busStopCode == "" || serviceNo == "" {
		http.Error(w, "BusStopCode and ServiceNo cannot be empty", http.StatusBadRequest)
		return
	}

	// Generate 3 random integers
	rand.Seed(time.Now().UnixNano())
	randomInts := make([]int, 3)
	for i := range randomInts {
		randomInts[i] = rand.Intn(1000) // Random numbers between 0-999
	}

	// Encode the array as JSON and send response
	if err := json.NewEncoder(w).Encode(randomInts); err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}

	slog.Info("S",
		"BusStopCode", busStopCode,
		"ServiceNo", serviceNo,
		"path", r.URL.Path,
		"method", r.Method,
		"client_ip", r.RemoteAddr,
	)
}

// CORS middleware function
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	}
}
