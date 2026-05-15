package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aattwwss/yabatasg/internal/handler"
	"github.com/aattwwss/yabatasg/internal/lta"
	"github.com/aattwwss/yabatasg/internal/store"
	"github.com/aattwwss/yabatasg/internal/syncer"
	"github.com/joho/godotenv"
)

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFiles embed.FS

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, using environment variables", "error", err)
	}

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		slog.Error("Failed to create static filesystem", "error", err)
		os.Exit(1)
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/yabatasg.db"
	}

	stopsStore, err := store.New(dbPath)
	if err != nil {
		slog.Error("Failed to open SQLite store", "path", dbPath, "error", err)
		os.Exit(1)
	}
	defer stopsStore.Close()

	indexTmpl, err := template.ParseFS(templateFiles, "templates/index.html")
	if err != nil {
		slog.Error("Template parsing failed", "error", err)
		os.Exit(1)
	}

	styleHash, _ := fileHash(staticFS, "style.css")
	scriptHash, _ := fileHash(staticFS, "script.js")
	manifestHash, _ := fileHash(staticFS, "manifest.json")
	iconSVGHash, _ := fileHash(staticFS, "icon.svg")
	icon180Hash, _ := fileHash(staticFS, "icon-180.png")
	swHash, _ := fileHash(templateFiles, "templates/sw.js")

	tmplHashes := map[string]string{
		"StyleCSS":    styleHash,
		"ScriptJS":    scriptHash,
		"Manifest":    manifestHash,
		"IconSVG":     iconSVGHash,
		"Icon180":     icon180Hash,
		"SWJS":        swHash,
	}

	ltaClient := lta.New(os.Getenv("LTA_ACCESS_KEY"), os.Getenv("LTA_API_HOST"))
	stopsSyncer := syncer.New(stopsStore, ltaClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go stopsSyncer.Run(ctx)

	mux := http.NewServeMux()

	mux.Handle("GET /static/", cacheStatic(http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))))

	swJS, _ := templateFiles.ReadFile("templates/sw.js")
	mux.HandleFunc("GET /sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		w.Write(swJS)
	})

	mux.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("User-agent: *\nAllow: /\nSitemap: https://yabatasg.com/sitemap.xml\n"))
	})

	mux.HandleFunc("GET /sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>https://yabatasg.com/</loc>
		<changefreq>weekly</changefreq>
		<priority>1.0</priority>
	</url>
</urlset>`))
	})

	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		if err := indexTmpl.Execute(w, tmplHashes); err != nil {
			slog.Error("Template execution failed", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
	mux.HandleFunc("GET /", serveIndex)
	mux.HandleFunc("GET /nearby", serveIndex)
	mux.HandleFunc("GET /stop/{code}", serveIndex)

	arrivalHandler := handler.NewBusArrival(ltaClient)
	mux.Handle("GET /api/v1/busArrival", corsMiddleware(arrivalHandler))

	nearbyHandler := handler.NewNearby(stopsStore)
	mux.Handle("GET /api/v1/stops/nearby", corsMiddleware(nearbyHandler))

	stopDetailHandler := handler.NewStopDetail(ltaClient)
	mux.Handle("GET /api/v1/stops/{code}/arrivals", corsMiddleware(stopDetailHandler))

	mux.HandleFunc("GET /api/v1/stops/{code}", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		code := r.PathValue("code")
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "stop code is required"})
			return
		}
		stop, err := stopsStore.GetStop(code)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		if stop == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "stop not found"})
			return
		}
		json.NewEncoder(w).Encode(stop)
	})))

	authHandler := handler.NewAuth(stopsStore)
	mux.Handle("POST /api/v1/auth/register", corsMiddleware(http.HandlerFunc(authHandler.Register)))
	mux.Handle("POST /api/v1/auth/link", corsMiddleware(http.HandlerFunc(authHandler.Link)))
	mux.Handle("GET /api/v1/auth/me", corsMiddleware(http.HandlerFunc(authHandler.Me)))

	configHandler := handler.NewConfig(stopsStore)
	mux.Handle("GET /api/v1/config", corsMiddleware(http.HandlerFunc(configHandler.Get)))
	mux.Handle("PUT /api/v1/config", corsMiddleware(http.HandlerFunc(configHandler.Put)))
	mux.Handle("DELETE /api/v1/config", corsMiddleware(http.HandlerFunc(configHandler.Delete)))

	mux.HandleFunc("POST /api/v1/stops/sync", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := stopsSyncer.SyncNow(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		slog.Info("Shutting down server...")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("Server forced to shutdown", "error", err)
		}
	}()

	slog.Info("Starting server", "port", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
	slog.Info("Server stopped")
}

func fileHash(fsys fs.FS, name string) (string, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:12]), nil
}

func cacheStatic(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		next.ServeHTTP(w, r)
	}
}

func corsMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
}
