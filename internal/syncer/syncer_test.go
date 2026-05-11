package syncer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aattwwss/yabatasg/internal/lta"
	"github.com/aattwwss/yabatasg/internal/store"
)

type mockClient struct {
	stops []lta.BusStop
}

func (m *mockClient) GetBusStops(ctx context.Context, skip int) (*lta.Response[lta.BusStop], error) {
	if skip >= len(m.stops) {
		return &lta.Response[lta.BusStop]{Value: []lta.BusStop{}}, nil
	}
	end := skip + 500
	if end > len(m.stops) {
		end = len(m.stops)
	}
	return &lta.Response[lta.BusStop]{Value: m.stops[skip:end]}, nil
}

func TestSyncNow(t *testing.T) {
	dbPath := "test_syncer.db"
	defer os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	client := &mockClient{
		stops: []lta.BusStop{
			{BusStopCode: "S1", RoadName: "Road 1", Description: "Desc 1", Latitude: 1.3, Longitude: 103.8},
			{BusStopCode: "S2", RoadName: "Road 2", Description: "Desc 2", Latitude: 1.31, Longitude: 103.81},
		},
	}

	syncer := New(s, client)
	if err := syncer.SyncNow(); err != nil {
		t.Fatalf("SyncNow failed: %v", err)
	}

	last, err := s.LastSynced()
	if err != nil {
		t.Fatal(err)
	}
	if last.IsZero() {
		t.Error("expected non-zero last_synced after sync")
	}

	results, err := s.Nearby(1.3, 103.8, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 stops, got %d", len(results))
	}
}

func TestSyncNowPagination(t *testing.T) {
	dbPath := "test_sync_pages.db"
	defer os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	// Create 600 stops (should trigger 2 pages: 500 + 100)
	stops := make([]lta.BusStop, 600)
	for i := range stops {
		stops[i] = lta.BusStop{
			BusStopCode: "S" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10)) + string(rune('0'+(i/100)%10)),
			RoadName:    "R", Description: "D",
			Latitude: 1.3, Longitude: 103.8,
		}
	}

	client := &mockClient{stops: stops}
	syncer := New(s, client)

	start := time.Now()
	if err := syncer.SyncNow(); err != nil {
		t.Fatalf("SyncNow failed: %v", err)
	}
	t.Logf("Synced 600 stops in %v", time.Since(start))

	// verify all 600 stops were inserted (they're all at same coordinates)
	results, err := s.Nearby(1.3, 103.8, 600)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 600 {
		t.Errorf("expected 600 stops, got %d", len(results))
	}
}
