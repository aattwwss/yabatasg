package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aattwwss/yabatasg/internal/lta"
	"github.com/aattwwss/yabatasg/internal/store"
)

func TestServiceSearchHandler(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	routes := []lta.BusRoute{
		{ServiceNo: "5", Direction: 1, StopSequence: 1, BusStopCode: "S1", Distance: 0},
		{ServiceNo: "51", Direction: 1, StopSequence: 1, BusStopCode: "S1", Distance: 0},
		{ServiceNo: "188", Direction: 1, StopSequence: 1, BusStopCode: "S3", Distance: 0},
	}
	if err := s.SyncRoutes(routes); err != nil {
		t.Fatalf("SyncRoutes failed: %v", err)
	}
	if err := s.SeedServiceOperators(); err != nil {
		t.Fatalf("SeedServiceOperators failed: %v", err)
	}

	h := NewService(s)

	t.Run("search matching prefix", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/services/search?q=5", nil)
		rec := httptest.NewRecorder()
		h.Search(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var results []store.ServiceSearchResult
		if err := json.NewDecoder(rec.Body).Decode(&results); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d: %v", len(results), results)
		}
	})

	t.Run("empty query returns empty array", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/services/search?q=", nil)
		rec := httptest.NewRecorder()
		h.Search(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var results []store.ServiceSearchResult
		json.NewDecoder(rec.Body).Decode(&results)
		if len(results) != 0 {
			t.Errorf("expected 0 results for empty query, got %d", len(results))
		}
	})

	t.Run("no match returns empty array", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/services/search?q=999", nil)
		rec := httptest.NewRecorder()
		h.Search(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var results []store.ServiceSearchResult
		json.NewDecoder(rec.Body).Decode(&results)
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})
}

func TestServiceStopsHandler(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	routes := []lta.BusRoute{
		{ServiceNo: "5", Direction: 1, StopSequence: 1, BusStopCode: "S1", Distance: 0},
		{ServiceNo: "5", Direction: 1, StopSequence: 2, BusStopCode: "S2", Distance: 1.2},
	}
	if err := s.SyncRoutes(routes); err != nil {
		t.Fatalf("SyncRoutes failed: %v", err)
	}

	h := NewService(s)

	t.Run("stops for known service", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/services/5/stops", nil)
		req.SetPathValue("no", "5")
		rec := httptest.NewRecorder()
		h.Stops(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var stops []store.ServiceStop
		if err := json.NewDecoder(rec.Body).Decode(&stops); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if len(stops) != 2 {
			t.Fatalf("expected 2 stops, got %d", len(stops))
		}
		if stops[0].StopCode != "S1" {
			t.Errorf("expected S1 first, got %s", stops[0].StopCode)
		}
	})

	t.Run("unknown service returns empty array", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/services/999/stops", nil)
		req.SetPathValue("no", "999")
		rec := httptest.NewRecorder()
		h.Stops(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var stops []store.ServiceStop
		json.NewDecoder(rec.Body).Decode(&stops)
		if len(stops) != 0 {
			t.Errorf("expected 0 stops, got %d", len(stops))
		}
	})

	t.Run("missing service number", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/services//stops", nil)
		rec := httptest.NewRecorder()
		h.Stops(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}
