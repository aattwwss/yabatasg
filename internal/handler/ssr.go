package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
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

	JSONLD template.HTML

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
func BuildStopJSONLD(stop *StopRenderData) template.HTML {
	var sb strings.Builder
	sb.WriteString(`{"@context":"https://schema.org","@type":"BusStop",`)
	fmt.Fprintf(&sb, `"name":"Stop %s — %s",`, stop.Code, stop.RoadName)
	fmt.Fprintf(&sb, `"identifier":"%s",`, stop.Code)
	fmt.Fprintf(&sb, `"url":"https://yabatasg.com/stop/%s",`, stop.Code)
	fmt.Fprintf(&sb, `"description":"Real-time bus arrival times for Stop %s (%s) in Singapore. Powered by LTA DataMall.",`, stop.Code, stop.RoadName)

	// containedInPlace
	sb.WriteString(`"containedInPlace":{"@type":"City","name":"Singapore"},`)

	// containsOffer — each bus service as a BusTrip
	sb.WriteString(`"containsOffer":[`)
	for i, svc := range stop.Services {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`{"@type":"BusTrip",`)
		fmt.Fprintf(&sb, `"name":"Bus %s",`, svc.ServiceNumber)
		if svc.Operator != "" {
			fmt.Fprintf(&sb, `"provider":{"@type":"Organization","name":"%s"},`, svc.Operator)
		}
		// next arrival as departureTime if available
		if svc.Next1 != nil && *svc.Next1 >= 0 {
			fmt.Fprintf(&sb, `"description":"Next bus in %d min"`, *svc.Next1)
		}
		sb.WriteString("}")
	}
	sb.WriteString("]}")

	return template.HTML(sb.String())
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
