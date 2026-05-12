package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
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

	sort.Slice(resp.Services, func(i, j int) bool {
		return serviceLess(resp.Services[i].ServiceNumber, resp.Services[j].ServiceNumber)
	})

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Error encoding response", "error", err)
	}
}

func serviceLess(a, b string) bool {
	aNum, aSfx := splitService(a)
	bNum, bSfx := splitService(b)
	if aNum != bNum {
		return aNum < bNum
	}
	return aSfx < bSfx
}

func splitService(s string) (int, string) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, s
	}
	return atoi(s[:i]), s[i:]
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}
