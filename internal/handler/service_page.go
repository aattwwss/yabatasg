package handler

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/aattwwss/yabatasg/internal/store"
)

// ServicePage serves server-rendered bus route pages at /service/{no}.
type ServicePage struct {
	store *store.Store
	base  TemplateData
	tmpl  *template.Template
}

func NewServicePage(s *store.Store, base TemplateData, tmpl *template.Template) *ServicePage {
	return &ServicePage{store: s, base: base, tmpl: tmpl}
}

func (h *ServicePage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serviceNo := r.PathValue("no")
	if serviceNo == "" {
		http.NotFound(w, r)
		return
	}

	stops, err := h.store.GetStopsByService(serviceNo)
	if err != nil {
		slog.Error("Failed to get stops by service", "serviceNo", serviceNo, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if len(stops) == 0 {
		http.NotFound(w, r)
		return
	}

	routeStops := make([]ServiceRouteStop, len(stops))
	directionLabels := make(map[int]string)
	var firstDirection int
	for i, s := range stops {
		routeStops[i] = ServiceRouteStop{
			Code:        s.StopCode,
			RoadName:    s.RoadName,
			Description: s.Description,
			Direction:   s.Direction,
			Sequence:    s.Sequence,
		}
		if firstDirection == 0 {
			firstDirection = s.Direction
		}
		label := s.Description
		if label == "" {
			label = s.RoadName
		}
		if label == "" {
			label = s.StopCode
		}
		directionLabels[s.Direction] = "To " + label
	}

	data := h.base
	data.Stop = nil

	srData := &ServiceRouteRenderData{
		ServiceNo:       serviceNo,
		Stops:           routeStops,
		FirstDirection:  firstDirection,
		DirectionLabels: directionLabels,
	}
	data.ServiceRoute = srData

	titleStop := fmt.Sprintf("Bus %s Route", serviceNo)
	data.Title = fmt.Sprintf("%s — Stops & Timings | yabata Singapore", titleStop)
	data.Description = fmt.Sprintf("See all bus stops for service %s in Singapore. View the full route, find your nearest stop, and check real-time arrival times. Powered by LTA DataMall.", serviceNo)
	data.Canonical = fmt.Sprintf("https://yabatasg.com/service/%s", serviceNo)
	data.OGTitle = fmt.Sprintf("%s | yabata", titleStop)
	data.OGDescription = fmt.Sprintf("Bus %s route stops in Singapore. Powered by LTA DataMall.", serviceNo)
	data.OGURL = data.Canonical
	data.JSONLD = BuildServiceRouteJSONLD(serviceNo)

	initState, err := BuildServiceInitialState(srData)
	if err != nil {
		slog.Warn("Failed to marshal service initial state", "code", serviceNo, "error", err)
	} else {
		data.InitialState = initState
	}

	if err := h.tmpl.Execute(w, data); err != nil {
		slog.Error("Template execution failed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
