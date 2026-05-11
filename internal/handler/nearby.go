package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/aattwwss/yabatasg/internal/store"
)

type Nearby struct {
	store *store.Store
}

func NewNearby(s *store.Store) *Nearby {
	return &Nearby{store: s}
}

func (h *Nearby) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := r.URL.Query()
	latStr := q.Get("lat")
	lngStr := q.Get("lng")
	limitStr := q.Get("limit")

	if latStr == "" || lngStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "lat and lng are required"})
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid lat"})
		return
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid lng"})
		return
	}

	limit := 15
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}

	stops, err := h.store.Nearby(lat, lng, limit)
	if err != nil {
		slog.Error("Error querying nearby stops", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to query nearby stops"})
		return
	}

	if stops == nil {
		stops = []store.StopWithDistance{}
	}

	if err := json.NewEncoder(w).Encode(stops); err != nil {
		slog.Error("Error encoding response", "error", err)
	}
}
