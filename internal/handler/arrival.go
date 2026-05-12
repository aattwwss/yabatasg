package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/aattwwss/yabatasg/internal/lta"
)

type LTAClient interface {
	GetBusArrival(ctx context.Context, busStopCode, serviceNumber string) (*lta.BusArrival, error)
}

type BusArrival struct {
	lta LTAClient
}

func NewBusArrival(client LTAClient) *BusArrival {
	return &BusArrival{lta: client}
}

func (h *BusArrival) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	busStopCode := r.URL.Query().Get("BusStopCode")
	serviceNo := r.URL.Query().Get("ServiceNo")

	if busStopCode == "" || serviceNo == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "BusStopCode and ServiceNo are required"})
		return
	}

	arrivals, err := h.lta.GetBusArrival(r.Context(), busStopCode, serviceNo)
	if err != nil {
		slog.Error("Error getting bus arrival from LTA API", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch arrival data"})
		return
	}

	res := [3]*int{}
	now := time.Now()
	for _, service := range arrivals.Services {
		if service.ServiceNumber == serviceNo {
			res[0] = ptr(diffMinutes(service.NextBus.EstimatedArrival.Time, now))
			res[1] = ptr(diffMinutes(service.NextBus2.EstimatedArrival.Time, now))
			res[2] = ptr(diffMinutes(service.NextBus3.EstimatedArrival.Time, now))
		}
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("Error encoding JSON response", "error", err)
	}
}

func diffMinutes(a, b time.Time) int {
	return int(a.Sub(b).Minutes())
}

func ptr[T any](v T) *T {
	return &v
}
