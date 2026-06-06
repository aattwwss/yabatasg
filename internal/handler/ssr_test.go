package handler

import (
	"encoding/json"
	"html/template"
	"testing"
)

func TestBuildStopJSONLD(t *testing.T) {
	stop := &StopRenderData{
		Code:        "12345",
		RoadName:    "Jln Eunos",
		Description: "Opp Eunos Stn",
		Services: []ServiceTiming{
			{ServiceNumber: "10", Operator: "SBST", Next1: intPtr(5)},
			{ServiceNumber: "196", Operator: "SMRT", Next1: intPtr(12)},
			{ServiceNumber: "15", Operator: "", Next1: nil},
		},
	}

	result := BuildStopJSONLD(stop)

	// Verify return type is template.JS.
	var _ template.JS = result

	// Verify the output is a JSON object, not a stringified string.
	if len(result) == 0 || result[0] != '{' {
		t.Fatalf("output does not start with '{', likely double-stringified: %s", result)
	}

	// Verify it's valid JSON.
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, result)
	}

	// Spot-check key fields.
	if parsed["@context"] != "https://schema.org" {
		t.Errorf("expected @context https://schema.org, got %v", parsed["@context"])
	}
	if parsed["@type"] != "BusStop" {
		t.Errorf("expected @type BusStop, got %v", parsed["@type"])
	}
	if parsed["identifier"] != "12345" {
		t.Errorf("expected identifier 12345, got %v", parsed["identifier"])
	}

	offers, ok := parsed["containsOffer"].([]interface{})
	if !ok {
		t.Fatalf("containsOffer is not an array, got %T", parsed["containsOffer"])
	}
	if len(offers) != 3 {
		t.Errorf("expected 3 offers, got %d", len(offers))
	}
}

func TestBuildStopJSONLDNoServices(t *testing.T) {
	stop := &StopRenderData{
		Code:     "54321",
		RoadName: "Upp Changi Rd",
		Services: nil,
	}

	result := BuildStopJSONLD(stop)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	offers, ok := parsed["containsOffer"].([]interface{})
	if !ok {
		t.Fatalf("containsOffer is not an array")
	}
	if len(offers) != 0 {
		t.Errorf("expected 0 offers, got %d", len(offers))
	}
}

func TestBuildStopJSONLDWithGeo(t *testing.T) {
	stop := &StopRenderData{
		Code:        "12345",
		RoadName:    "Jln Eunos",
		Description: "Opp Eunos Stn",
		Latitude:    1.3136,
		Longitude:   103.8925,
		Services:    nil,
	}

	result := BuildStopJSONLD(stop)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	geo, ok := parsed["geo"].(map[string]any)
	if !ok {
		t.Fatal("geo field missing or not an object")
	}
	if geo["@type"] != "GeoCoordinates" {
		t.Errorf("expected GeoCoordinates, got %v", geo["@type"])
	}
	if geo["latitude"] != 1.3136 {
		t.Errorf("expected latitude 1.3136, got %v", geo["latitude"])
	}
	if geo["longitude"] != 103.8925 {
		t.Errorf("expected longitude 103.8925, got %v", geo["longitude"])
	}

	addr, ok := parsed["address"].(map[string]any)
	if !ok {
		t.Fatal("address field missing or not an object")
	}
	if addr["@type"] != "PostalAddress" {
		t.Errorf("expected PostalAddress, got %v", addr["@type"])
	}
	if addr["addressCountry"] != "SG" {
		t.Errorf("expected SG, got %v", addr["addressCountry"])
	}
}

func TestBuildStopJSONLDNoGeo(t *testing.T) {
	stop := &StopRenderData{
		Code:     "12345",
		RoadName: "Jln Eunos",
		Services: nil,
	}

	result := BuildStopJSONLD(stop)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if _, ok := parsed["geo"]; ok {
		t.Error("geo should be omitted when lat/lng are zero")
	}
	if _, ok := parsed["address"]; ok {
		t.Error("address should be omitted when lat/lng are zero")
	}
}

func TestBuildHomeJSONLD(t *testing.T) {
	result := BuildHomeJSONLD()

	if len(result) == 0 || result[0] != '{' {
		t.Fatalf("output does not start with '{': %s", result)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if parsed["@context"] != "https://schema.org" {
		t.Errorf("expected @context https://schema.org, got %v", parsed["@context"])
	}
	if parsed["@type"] != "WebSite" {
		t.Errorf("expected @type WebSite, got %v", parsed["@type"])
	}

	pa, ok := parsed["potentialAction"].(map[string]any)
	if !ok {
		t.Fatal("potentialAction missing or not an object")
	}
	if pa["@type"] != "SearchAction" {
		t.Errorf("expected SearchAction, got %v", pa["@type"])
	}
	if pa["target"] != "https://yabatasg.com/stop/{bus_stop_code}" {
		t.Errorf("unexpected target: %v", pa["target"])
	}
}

func TestBuildBreadcrumbJSONLD(t *testing.T) {
	result := BuildBreadcrumbJSONLD("12345", "Jln Eunos")

	if len(result) == 0 || result[0] != '{' {
		t.Fatalf("output does not start with '{': %s", result)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if parsed["@context"] != "https://schema.org" {
		t.Errorf("expected @context https://schema.org, got %v", parsed["@context"])
	}
	if parsed["@type"] != "BreadcrumbList" {
		t.Errorf("expected @type BreadcrumbList, got %v", parsed["@type"])
	}

	items, ok := parsed["itemListElement"].([]interface{})
	if !ok {
		t.Fatalf("itemListElement is not an array")
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	first := items[0].(map[string]any)
	if first["name"] != "Home" {
		t.Errorf("expected first item Home, got %v", first["name"])
	}

	second := items[1].(map[string]any)
	if second["name"] != "Bus Stop 12345 — Jln Eunos" {
		t.Errorf("unexpected second item name: %v", second["name"])
	}
}

func intPtr(v int) *int { return &v }
