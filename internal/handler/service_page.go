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
			Latitude:    s.Latitude,
			Longitude:   s.Longitude,
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

	// Extract origin (first stop) and destination (last stop) for the first direction.
	var originName, destName string
	var lastSeq int
	for _, s := range stops {
		if s.Direction == firstDirection {
			if s.Sequence == 1 {
				originName = stopLabel(s.Description, s.RoadName, s.StopCode)
			}
			if s.Sequence > lastSeq {
				lastSeq = s.Sequence
				destName = stopLabel(s.Description, s.RoadName, s.StopCode)
			}
		}
	}

	operator, err := h.store.GetServiceOperator(serviceNo)
	if err != nil {
		slog.Warn("Failed to get operator", "serviceNo", serviceNo, "error", err)
		operator = ""
	}

	srData := &ServiceRouteRenderData{
		ServiceNo:       serviceNo,
		Stops:           routeStops,
		FirstDirection:  firstDirection,
		DirectionLabels: directionLabels,
	}
	data.ServiceRoute = srData

	// SEO-optimized title: under 60 chars, includes bus number + "Route Singapore".
	data.Title = fmt.Sprintf("Bus %s Route Singapore | Full Stops & Schedule", serviceNo)
	// Description under 160 chars, including origin/destination when available.
	if originName != "" && destName != "" {
		data.Description = fmt.Sprintf("Find the full Bus %s route from %s to %s. All stops with live arrivals and schedule. Powered by LTA DataMall.", serviceNo, originName, destName)
	} else {
		data.Description = fmt.Sprintf("Find the full Bus %s route in Singapore. View all stops, check real-time arrival times, and plan your journey.", serviceNo)
	}
	data.Canonical = fmt.Sprintf("https://yabatasg.com/service/%s", serviceNo)
	data.OGTitle = fmt.Sprintf("Bus %s Route | Stops & Schedule, Singapore", serviceNo)
	data.OGDescription = data.Description
	data.OGURL = data.Canonical

	data.JSONLD = BuildServiceRouteJSONLD(srData, operator)

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

// stopLabel returns a human-readable label for a stop, preferring
// Description over RoadName over StopCode.
func stopLabel(desc, road, code string) string {
	if desc != "" {
		return desc
	}
	if road != "" {
		return road
	}
	return code
}
