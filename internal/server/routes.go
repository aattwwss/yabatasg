package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/aattwwss/yabatasg/cmd/web"
)

func (s *Server) RegisterRoutes() http.Handler {

	mux := http.NewServeMux()
	mux.HandleFunc("GET /busService", s.BusService)
	mux.HandleFunc("GET /", s.HelloWorldHandler)

	mux.HandleFunc("GET /health", s.healthHandler)

	fileServer := http.FileServer(http.FS(web.Files))
	mux.Handle("GET /assets/", fileServer)
	mux.Handle("GET /web", templ.Handler(web.HelloForm()))
	mux.HandleFunc("GET /hello", web.HelloWebHandler)

	s.scheduler.RegisterRoutes(mux)
	return mux
}

func (s *Server) CrawlBusService(ctx context.Context) {
}

func (s *Server) BusService(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	skip, _ := strconv.Atoi((r.URL.Query().Get("$skip")))
	busServices, _ := s.ltaAPIClient.GetBusServices(r.Context(), skip)
	json, _ := json.Marshal(busServices)
	w.Write(json)
}

func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	jsonResp, err := json.Marshal(s.db.Health())

	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}
