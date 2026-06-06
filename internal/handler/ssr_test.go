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

func intPtr(v int) *int { return &v }
