package handler

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/aattwwss/yabatasg/internal/store"
)

// StopPage serves server-rendered bus stop pages at /stop/{code}.
type StopPage struct {
	store *store.Store
	lta   LTAClient
	base  TemplateData
	tmpl  *template.Template
}

func NewStopPage(s *store.Store, lta LTAClient, base TemplateData, tmpl *template.Template) *StopPage {
	return &StopPage{store: s, lta: lta, base: base, tmpl: tmpl}
}

func (h *StopPage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	stop, err := h.store.GetStop(code)
	if err != nil {
		slog.Error("Failed to get stop", "code", code, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if stop == nil {
		http.NotFound(w, r)
		return
	}

	data := h.base
	now := time.Now()
	var services []ServiceTiming

	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()
	arrivals, err := h.lta.GetBusArrival(ctx, code, "")
	if err == nil {
		for _, svc := range arrivals.Services {
			services = append(services, ServiceTiming{
				ServiceNumber: svc.ServiceNumber,
				Operator:      svc.Operator,
				Next1:         new(DiffMinutes(svc.NextBus.EstimatedArrival.Time, now)),
				Next2:         new(DiffMinutes(svc.NextBus2.EstimatedArrival.Time, now)),
				Next3:         new(DiffMinutes(svc.NextBus3.EstimatedArrival.Time, now)),
			})
		}
	} else {
		slog.Warn("Failed to fetch arrivals for SSR", "code", code, "error", err)
	}

	data.Stop = &StopRenderData{
		Code:        stop.Code,
		RoadName:    stop.RoadName,
		Description: stop.Description,
		Services:    services,
	}

	data.Title = fmt.Sprintf("Bus Stop %s — %s | yabata Singapore", code, stop.RoadName)
	data.Description = fmt.Sprintf("Real-time bus arrival times for Stop %s (%s), Singapore. Check live next-bus timings for all services at this stop. Powered by LTA DataMall.", code, stop.RoadName)
	data.Canonical = fmt.Sprintf("https://yabatasg.com/stop/%s", code)
	data.OGTitle = fmt.Sprintf("Bus Stop %s — %s | yabata", code, stop.RoadName)
	data.OGDescription = fmt.Sprintf("Live bus arrivals for Stop %s (%s), Singapore. Powered by LTA DataMall.", code, stop.RoadName)
	data.OGURL = data.Canonical
	data.JSONLD = BuildStopJSONLD(data.Stop)

	initState, err := BuildInitialState(data.Stop)
	if err != nil {
		slog.Warn("Failed to marshal initial state", "code", code, "error", err)
	} else {
		data.InitialState = initState
	}

	if err := h.tmpl.Execute(w, data); err != nil {
		slog.Error("Template execution failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
