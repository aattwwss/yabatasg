package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aattwwss/yabatasg/internal/lta"
)

type mockLTA struct{}

func (m *mockLTA) GetBusArrival(ctx context.Context, busStopCode, serviceNumber string) (*lta.BusArrival, error) {
	return &lta.BusArrival{
		BusStopCode: "12345",
		Services: []lta.Service{
			{ServiceNumber: "10", Operator: "SBST"},
			{ServiceNumber: "196", Operator: "SMRT"},
		},
	}, nil
}

func TestStopDetailHandler(t *testing.T) {
	h := NewStopDetail(&mockLTA{})

	req := httptest.NewRequest("GET", "/api/v1/stops/12345/arrivals", nil)
	req.SetPathValue("code", "12345")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp StopArrivalResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.BusStopCode != "12345" {
		t.Errorf("expected 12345, got %s", resp.BusStopCode)
	}
	if len(resp.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(resp.Services))
	}
}

func TestServiceLess(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"10", "51", true},
		{"51", "10", false},
		{"196", "851", true},
		{"851", "196", false},
		{"10", "10e", true},
		{"10e", "196", true},
		{"851", "851e", true},
		{"851e", "851", false},
	}
	for _, tt := range tests {
		got := serviceLess(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("serviceLess(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestStopDetailHandlerMissingCode(t *testing.T) {
	h := NewStopDetail(&mockLTA{})

	req := httptest.NewRequest("GET", "/api/v1/stops//arrivals", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
