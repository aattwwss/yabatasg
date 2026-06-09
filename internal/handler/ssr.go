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

	JSONLD       template.JS
	BreadcrumbLD template.JS

	Stop *StopRenderData

	ServiceRoute *ServiceRouteRenderData

	InitialState template.JS
}

// StopRenderData carries stop info and arrival data for SSR and initial state hydration.
type StopRenderData struct {
	Code        string          `json:"code"`
	RoadName    string          `json:"roadName"`
	Description string          `json:"description"`
	Latitude    float64         `json:"latitude"`
	Longitude   float64         `json:"longitude"`
	Services    []ServiceTiming `json:"services"`
}

// ServiceRouteRenderData carries bus route data for SSR and initial state hydration.
type ServiceRouteRenderData struct {
	ServiceNo       string             `json:"serviceNo"`
	Stops           []ServiceRouteStop `json:"stops"`
	FirstDirection  int                `json:"-"`
	DirectionLabels map[int]string     `json:"-"`
}

// ServiceRouteStop is a single stop along a bus route.
type ServiceRouteStop struct {
	Code        string `json:"stopCode"`
	RoadName    string `json:"roadName"`
	Description string `json:"description"`
	Direction   int    `json:"direction"`
	Sequence    int    `json:"sequence"`
}

// BuildHomeJSONLD returns a WebSite + SearchAction JSON-LD script for the homepage.
func BuildHomeJSONLD() template.JS {
	type potentialAction struct {
		Type       string `json:"@type"`
		Target     string `json:"target"`
		QueryInput string `json:"query-input"`
	}
	type website struct {
		Context         string           `json:"@context"`
		Type            string           `json:"@type"`
		Name            string           `json:"name"`
		URL             string           `json:"url"`
		Description     string           `json:"description"`
		PotentialAction *potentialAction `json:"potentialAction,omitempty"`
	}

	b, _ := json.Marshal(website{
		Context:     "https://schema.org",
		Type:        "WebSite",
		Name:        "Singapore Bus Arrival Times — yabata",
		URL:         "https://yabatasg.com",
		Description: "Real-time bus arrival times for every bus stop in Singapore. Powered by LTA DataMall.",
		PotentialAction: &potentialAction{
			Type:       "SearchAction",
			Target:     "https://yabatasg.com/stop/{bus_stop_code}",
			QueryInput: "required name=bus_stop_code",
		},
	})
	return template.JS(b)
}

// BuildStopJSONLD constructs a JSON-LD script body for a bus stop with BusTrip offers.
func BuildStopJSONLD(stop *StopRenderData) template.JS {
	type place struct {
		Type string `json:"@type"`
		Name string `json:"name"`
	}
	type geo struct {
		Type      string  `json:"@type"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	type address struct {
		Type            string `json:"@type"`
		StreetAddress   string `json:"streetAddress"`
		AddressLocality string `json:"addressLocality"`
		AddressCountry  string `json:"addressCountry"`
	}
	type trip struct {
		Type        string `json:"@type"`
		Name        string `json:"name"`
		Provider    *place `json:"provider,omitempty"`
		Description string `json:"description,omitempty"`
	}
	type ld struct {
		Context          string   `json:"@context"`
		Type             string   `json:"@type"`
		Name             string   `json:"name"`
		Identifier       string   `json:"identifier"`
		URL              string   `json:"url"`
		Description      string   `json:"description"`
		ContainedInPlace place    `json:"containedInPlace"`
		Geo              *geo     `json:"geo,omitempty"`
		Address          *address `json:"address,omitempty"`
		ContainsOffer    []trip   `json:"containsOffer"`
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

	name := fmt.Sprintf("Bus Stop %s — %s", stop.Code, stop.RoadName)
	if stop.Description != "" && stop.Description != stop.RoadName {
		name = fmt.Sprintf("Bus Stop %s — %s (%s)", stop.Code, stop.Description, stop.RoadName)
	}
	desc := fmt.Sprintf("Real-time bus arrival times for Bus Stop %s (%s) in Singapore. Powered by LTA DataMall.", stop.Code, stop.RoadName)
	if stop.Description != "" {
		desc = fmt.Sprintf("Real-time bus arrival times for Bus Stop %s — %s (%s) in Singapore. Powered by LTA DataMall.", stop.Code, stop.Description, stop.RoadName)
	}

	result := ld{
		Context:     "https://schema.org",
		Type:        "BusStop",
		Name:        name,
		Identifier:  stop.Code,
		URL:         fmt.Sprintf("https://yabatasg.com/stop/%s", stop.Code),
		Description: desc,
		ContainedInPlace: place{
			Type: "City",
			Name: "Singapore",
		},
		ContainsOffer: trips,
	}
	if stop.Latitude != 0 || stop.Longitude != 0 {
		result.Geo = &geo{
			Type:      "GeoCoordinates",
			Latitude:  stop.Latitude,
			Longitude: stop.Longitude,
		}
		result.Address = &address{
			Type:            "PostalAddress",
			StreetAddress:   fmt.Sprintf("%s %s", stop.RoadName, stop.Code),
			AddressLocality: "Singapore",
			AddressCountry:  "SG",
		}
	}

	b, err := json.Marshal(result)
	if err != nil {
		return template.JS("")
	}
	return template.JS(b)
}

// BuildBreadcrumbJSONLD returns a BreadcrumbList JSON-LD for a stop page.
func BuildBreadcrumbJSONLD(code, roadName string) template.JS {
	type item struct {
		Type     string `json:"@type"`
		Position int    `json:"position"`
		Name     string `json:"name"`
		Item     string `json:"item"`
	}
	type breadcrumb struct {
		Context  string `json:"@context"`
		Type     string `json:"@type"`
		ItemList []item `json:"itemListElement"`
	}

	b, _ := json.Marshal(breadcrumb{
		Context: "https://schema.org",
		Type:    "BreadcrumbList",
		ItemList: []item{
			{Type: "ListItem", Position: 1, Name: "Home", Item: "https://yabatasg.com"},
			{Type: "ListItem", Position: 2, Name: fmt.Sprintf("Bus Stop %s — %s", code, roadName), Item: fmt.Sprintf("https://yabatasg.com/stop/%s", code)},
		},
	})
	return template.JS(b)
}

// BuildServiceRouteJSONLD constructs a JSON-LD script for a bus route page.
func BuildServiceRouteJSONLD(serviceNo string) template.JS {
	type ld struct {
		Context     string `json:"@context"`
		Type        string `json:"@type"`
		Name        string `json:"name"`
		Description string `json:"description"`
		URL         string `json:"url"`
	}

	b, _ := json.Marshal(ld{
		Context:     "https://schema.org",
		Type:        "BusTrip",
		Name:        fmt.Sprintf("Bus %s Route", serviceNo),
		Description: fmt.Sprintf("Bus route and stops for service %s in Singapore. Powered by LTA DataMall.", serviceNo),
		URL:         fmt.Sprintf("https://yabatasg.com/service/%s", serviceNo),
	})
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

// BuildServiceInitialState serializes service route data for Alpine.js hydration.
func BuildServiceInitialState(sr *ServiceRouteRenderData) (template.JS, error) {
	data, err := json.Marshal(sr)
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
