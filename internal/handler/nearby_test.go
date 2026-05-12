package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aattwwss/yabatasg/internal/store"
)

func TestNearbyHandlerMissingParams(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	h := NewNearby(s)

	tests := []struct {
		name string
		url  string
	}{
		{"no params", "/api/v1/stops/nearby"},
		{"only lat", "/api/v1/stops/nearby?lat=1.3"},
		{"only lng", "/api/v1/stops/nearby?lng=103.8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rec.Code)
			}

			var resp map[string]string
			json.NewDecoder(rec.Body).Decode(&resp)
			if resp["error"] == "" {
				t.Error("expected error message")
			}
		})
	}
}

func TestNearbyHandlerSuccess(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	req := httptest.NewRequest("GET", "/api/v1/stops/nearby?lat=1.3&lng=103.8", nil)
	rec := httptest.NewRecorder()
	h := NewNearby(s)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var stops []store.StopWithDistance
	if err := json.NewDecoder(rec.Body).Decode(&stops); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	// empty DB should return empty array, not null
	if stops == nil {
		t.Error("expected empty array, got nil")
	}
}

func TestNearbyHandlerInvalidParams(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	req := httptest.NewRequest("GET", "/api/v1/stops/nearby?lat=abc&lng=103.8", nil)
	rec := httptest.NewRecorder()
	NewNearby(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
