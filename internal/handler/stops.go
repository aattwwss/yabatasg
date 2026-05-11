package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/aattwwss/yabatasg/internal/lta"
)

type StopDetailClient interface {
	GetBusArrival(ctx context.Context, busStopCode, serviceNumber string) (*lta.BusArrival, error)
}

type StopDetail struct {
	lta StopDetailClient
}

func NewStopDetail(client StopDetailClient) *StopDetail {
	return &StopDetail{lta: client}
}

type StopArrivalResponse struct {
	BusStopCode string          `json:"busStopCode"`
	Services    []ServiceTiming `json:"services"`
}

type ServiceTiming struct {
	ServiceNumber string `json:"serviceNo"`
	Operator      string `json:"operator"`
	Next1         *int   `json:"next1"`
	Next2         *int   `json:"next2"`
	Next3         *int   `json:"next3"`
}

func (h *StopDetail) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	code := r.PathValue("code")
	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "stop code is required"})
		return
	}

	arrivals, err := h.lta.GetBusArrival(r.Context(), code, "")
	if err != nil {
		slog.Error("Error getting bus arrivals for stop", "code", code, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch arrivals"})
		return
	}

	resp := StopArrivalResponse{
		BusStopCode: arrivals.BusStopCode,
	}

	now := time.Now()
	for _, svc := range arrivals.Services {
		resp.Services = append(resp.Services, ServiceTiming{
			ServiceNumber: svc.ServiceNumber,
			Operator:      svc.Operator,
			Next1:         ptr(diffMinutes(svc.NextBus.EstimatedArrival.Time, now)),
			Next2:         ptr(diffMinutes(svc.NextBus2.EstimatedArrival.Time, now)),
			Next3:         ptr(diffMinutes(svc.NextBus3.EstimatedArrival.Time, now)),
		})
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Error encoding response", "error", err)
	}
}
