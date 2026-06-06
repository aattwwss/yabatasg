package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
)

// TemplateData is passed to every HTML template execution.
type TemplateData struct {
	StyleCSS string
	ScriptJS string
	Manifest string
	IconSVG  string
	Icon180  string
	SWJS     string

	Title         string
	Description   string
	Canonical     string
	OGTitle       string
	OGDescription string
	OGURL         string

	JSONLD template.JS

	Stop *StopRenderData

	PopularStops []PopularStop

	InitialState template.JS
}

// StopRenderData carries stop info and arrival data for SSR and initial state hydration.
type StopRenderData struct {
	Code        string          `json:"code"`
	RoadName    string          `json:"roadName"`
	Description string          `json:"description"`
	Services    []ServiceTiming `json:"services"`
}

// PopularStop is a curated bus stop for the homepage crawlable links.
type PopularStop struct {
	Code     string
	RoadName string
}

// BuildStopJSONLD constructs a JSON-LD script body for a bus stop with BusTrip offers.
func BuildStopJSONLD(stop *StopRenderData) template.JS {
	type place struct {
		Type string `json:"@type"`
		Name string `json:"name"`
	}
	type trip struct {
		Type        string  `json:"@type"`
		Name        string  `json:"name"`
		Provider    *place  `json:"provider,omitempty"`
		Description string  `json:"description,omitempty"`
	}
	type ld struct {
		Context          string `json:"@context"`
		Type             string `json:"@type"`
		Name             string `json:"name"`
		Identifier       string `json:"identifier"`
		URL              string `json:"url"`
		Description      string `json:"description"`
		ContainedInPlace place  `json:"containedInPlace"`
		ContainsOffer    []trip `json:"containsOffer"`
	}

	trips := make([]trip, 0, len(stop.Services))
	for _, svc := range stop.Services {
		t := trip{
			Type: "BusTrip",
			Name: "Bus " + svc.ServiceNumber,
		}
		if svc.Operator != "" {
			t.Provider = &place{Type: "Organization", Name: svc.Operator}
		}
		if svc.Next1 != nil && *svc.Next1 >= 0 {
			t.Description = fmt.Sprintf("Next bus in %d min", *svc.Next1)
		}
		trips = append(trips, t)
	}

	b, err := json.Marshal(ld{
		Context:     "https://schema.org",
		Type:        "BusStop",
		Name:        fmt.Sprintf("Stop %s — %s", stop.Code, stop.RoadName),
		Identifier:  stop.Code,
		URL:         fmt.Sprintf("https://yabatasg.com/stop/%s", stop.Code),
		Description: fmt.Sprintf("Real-time bus arrival times for Stop %s (%s) in Singapore. Powered by LTA DataMall.", stop.Code, stop.RoadName),
		ContainedInPlace: place{
			Type: "City",
			Name: "Singapore",
		},
		ContainsOffer: trips,
	})
	if err != nil {
		return template.JS("")
	}
	return template.JS(b)
}

// BuildInitialState serializes stop data for Alpine.js hydration.
func BuildInitialState(stop *StopRenderData) (template.JS, error) {
	data, err := json.Marshal(stop)
	if err != nil {
		return "", err
	}
	return template.JS(data), nil
}

// FormatArrival formats arrival minutes for template rendering.
func FormatArrival(v *int) string {
	if v == nil || *v < 0 {
		return "--"
	}
	return fmt.Sprintf("%d", *v)
}

// ArrivalClass returns the CSS class for an arrival time value.
func ArrivalClass(v *int) string {
	if v == nil || *v < 0 {
		return ""
	}
	if *v <= 2 {
		return "urgent"
	}
	if *v <= 8 {
		return "soon"
	}
	return "later"
}
