package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/aattwwss/yabatasg/internal/store"
)

type Service struct {
	store *store.Store
}

func NewService(s *store.Store) *Service {
	return &Service{store: s}
}

func (h *Service) Search(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := r.URL.Query().Get("q")
	if q == "" {
		json.NewEncoder(w).Encode([]store.ServiceSearchResult{})
		return
	}

	results, err := h.store.SearchServices(q)
	if err != nil {
		slog.Error("Error searching services", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to search services"})
		return
	}

	if results == nil {
		results = []store.ServiceSearchResult{}
	}

	json.NewEncoder(w).Encode(results)
}

func (h *Service) Stops(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	serviceNo := r.PathValue("no")
	if serviceNo == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "service number is required"})
		return
	}

	stops, err := h.store.GetStopsByService(serviceNo)
	if err != nil {
		slog.Error("Error getting stops by service", "serviceNo", serviceNo, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get stops"})
		return
	}

	if stops == nil {
		stops = []store.ServiceStop{}
	}

	json.NewEncoder(w).Encode(stops)
}
