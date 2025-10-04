package main

import (
	"context"
	"embed"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"encoding/json"

	"github.com/aattwwss/yabatasg/ltaapi"
	"github.com/joho/godotenv"
)

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFiles embed.FS

func main() {
	// Configure structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	err := godotenv.Load()
	if err != nil {
		slog.Error("Error loading .env file", "error", err)
		os.Exit(1)
	}

	// Create HTTP handler for static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		slog.Error("Failed to create static filesystem", "error", err)
		os.Exit(1)
	}
	http.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Define route for root path
	http.HandleFunc("GET /", homeHandler)

	ltaClient := ltaapi.New(os.Getenv("LTA_ACCESS_KEY"), os.Getenv("LTA_API_HOST"))
	busArrivalHandler := busArrivalHandler{ltaClient: &ltaClient}

	http.HandleFunc("GET /api/v1/busArrival", corsMiddleware(busArrivalHandler.arrivalHandler))

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

type ltaClientInterface interface {
	GetBusArrival(ctx context.Context, busStopCode string, serviceNumber string) (*ltaapi.BusArrival, error)
}

type busArrivalHandler struct {
	ltaClient ltaClientInterface
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

func (ba busArrivalHandler) arrivalHandler(w http.ResponseWriter, r *http.Request) {
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

	arrivals, err := ba.ltaClient.GetBusArrival(context.Background(), busStopCode, serviceNo)
	if err != nil {
		slog.Error("Error getting bus arrival from lta api", "error", err)
	}

	res := [3]*int{}

	now := time.Now()
	for _, service := range arrivals.Services {
		if service.ServiceNumber == serviceNo {
			res[0] = Ptr(diffMinutes(service.NextBus.EstimatedArrival, now))
			res[1] = Ptr(diffMinutes(service.NextBus2.EstimatedArrival, now))
			res[2] = Ptr(diffMinutes(service.NextBus3.EstimatedArrival, now))
		}

	}

	// Encode the array as JSON and send response
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}

	slog.Info("received request to arrivalHandler",
		"BusStopCode", busStopCode,
		"ServiceNo", serviceNo,
		"path", r.URL.Path,
		"method", r.Method,
		"client_ip", r.RemoteAddr,
	)
}

func diffMinutes(a, b time.Time) int {
	return int(a.Sub(b).Minutes())
}

func Ptr[T any](v T) *T {
	return &v
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
