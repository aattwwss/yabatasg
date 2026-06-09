package store

import (
	"math"
	"testing"

	"github.com/aattwwss/yabatasg/internal/lta"
)

func TestHaversine(t *testing.T) {
	// Same point → 0 distance
	d := haversine(1.3, 103.8, 1.3, 103.8)
	if d != 0 {
		t.Errorf("expected 0, got %f", d)
	}

	// ~111 km per degree latitude
	d = haversine(0, 0, 1, 0)
	km := d / 1000
	if km < 110 || km > 112 {
		t.Errorf("expected ~111 km, got %f km", km)
	}

	// Known distance: Singapore CBD to Orchard (~3.3 km)
	d = haversine(1.2835, 103.8517, 1.3039, 103.8318)
	km = d / 1000
	if km < 3.0 || km > 3.8 {
		t.Errorf("expected ~3.3 km, got %f km", km)
	}
}

func TestStoreNearby(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	stops := []stop{
		{code: "S1", lat: 1.3000, lng: 103.8000},
		{code: "S2", lat: 1.3010, lng: 103.8010},
		{code: "S3", lat: 1.3150, lng: 103.8150}, // farther but within bounding box
	}

	tx, err := s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	stmt, err := tx.Prepare(`INSERT INTO bus_stops (code, road_name, description, latitude, longitude) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range stops {
		if _, err := stmt.Exec(st.code, "Road "+st.code, "Desc "+st.code, st.lat, st.lng); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	results, err := s.Nearby(1.3000, 103.8000, 10)
	if err != nil {
		t.Fatalf("Nearby failed: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	// S1 should be closest (same coordinates)
	if results[0].Code != "S1" {
		t.Errorf("expected S1 first, got %s", results[0].Code)
	}

	// S3 should be last (far away)
	if results[len(results)-1].Code != "S3" {
		t.Errorf("expected S3 last, got %s", results[len(results)-1].Code)
	}
}

func TestStoreNearbyLimit(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	tx, _ := s.db.Begin()
	stmt, _ := tx.Prepare(`INSERT INTO bus_stops (code, road_name, description, latitude, longitude) VALUES (?, ?, ?, ?, ?)`)
	for i := 0; i < 5; i++ {
		lat := 1.3000 + float64(i)*0.0001
		stmt.Exec("S"+string(rune('0'+i)), "R", "D", lat, 103.8000)
	}
	tx.Commit()

	results, err := s.Nearby(1.3000, 103.8000, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestLastSynced(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	ts, err := s.LastSynced()
	if err != nil {
		t.Fatal(err)
	}
	if !ts.IsZero() {
		t.Errorf("expected zero time for fresh DB, got %v", ts)
	}
}

func TestHaversineSymmetry(t *testing.T) {
	a := haversine(1.3, 103.8, 1.35, 103.85)
	b := haversine(1.35, 103.85, 1.3, 103.8)
	if math.Abs(a-b) > 0.001 {
		t.Errorf("haversine not symmetric: %f vs %f", a, b)
	}
}

func TestSearchServices(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	routes := []lta.BusRoute{
		{ServiceNo: "5", Direction: 1, StopSequence: 1, BusStopCode: "S1", Distance: 0},
		{ServiceNo: "5", Direction: 1, StopSequence: 2, BusStopCode: "S2", Distance: 1.2},
		{ServiceNo: "51", Direction: 1, StopSequence: 1, BusStopCode: "S1", Distance: 0},
		{ServiceNo: "188", Direction: 1, StopSequence: 1, BusStopCode: "S3", Distance: 0},
	}
	if err := s.SyncRoutes(routes); err != nil {
		t.Fatalf("SyncRoutes failed: %v", err)
	}
	if err := s.SeedServiceOperators(); err != nil {
		t.Fatalf("SeedServiceOperators failed: %v", err)
	}

	tests := []struct {
		query string
		want  int
	}{
		{"5", 2},     // 5, 51
		{"51", 1},    // 51
		{"188", 1},   // 188
		{"999", 0},   // none
		{"", 3},      // all (LIKE '%')
	}
	for _, tt := range tests {
		results, err := s.SearchServices(tt.query)
		if err != nil {
			t.Errorf("SearchServices(%q) error: %v", tt.query, err)
			continue
		}
		if len(results) != tt.want {
			t.Errorf("SearchServices(%q) = %d results, want %d", tt.query, len(results), tt.want)
		}
	}
}

func TestGetStopsByService(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	// Insert bus stops first (needed for the JOIN)
	tx, _ := s.db.Begin()
	stmt, _ := tx.Prepare(`INSERT INTO bus_stops (code, road_name, description, latitude, longitude) VALUES (?, ?, ?, ?, ?)`)
	stmt.Exec("S1", "Road 1", "Desc 1", 1.3, 103.8)
	stmt.Exec("S2", "Road 2", "Desc 2", 1.31, 103.81)
	tx.Commit()

	routes := []lta.BusRoute{
		{ServiceNo: "5", Direction: 1, StopSequence: 1, BusStopCode: "S1", Distance: 0},
		{ServiceNo: "5", Direction: 1, StopSequence: 2, BusStopCode: "S2", Distance: 1.2},
		{ServiceNo: "5", Direction: 2, StopSequence: 1, BusStopCode: "S2", Distance: 0},
	}
	if err := s.SyncRoutes(routes); err != nil {
		t.Fatalf("SyncRoutes failed: %v", err)
	}

	stops, err := s.GetStopsByService("5")
	if err != nil {
		t.Fatalf("GetStopsByService failed: %v", err)
	}
	if len(stops) != 3 {
		t.Fatalf("expected 3 stops, got %d", len(stops))
	}

	// Direction 1 stops should come first
	if stops[0].Direction != 1 || stops[0].Sequence != 1 {
		t.Errorf("expected dir 1 seq 1 first, got dir %d seq %d", stops[0].Direction, stops[0].Sequence)
	}
	if stops[0].RoadName != "Road 1" {
		t.Errorf("expected Road 1, got %s", stops[0].RoadName)
	}

	// Empty for unknown service
	empty, err := s.GetStopsByService("999")
	if err != nil {
		t.Fatalf("GetStopsByService(999) error: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 stops for unknown service, got %d", len(empty))
	}
}

func TestSyncRoutes(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	routes := []lta.BusRoute{
		{ServiceNo: "10", Direction: 1, StopSequence: 1, BusStopCode: "A1", Distance: 0},
		{ServiceNo: "10", Direction: 1, StopSequence: 2, BusStopCode: "A2", Distance: 1.5},
	}
	if err := s.SyncRoutes(routes); err != nil {
		t.Fatalf("first SyncRoutes failed: %v", err)
	}

	// Verify they exist
	stops, _ := s.GetStopsByService("10")
	if len(stops) != 2 {
		t.Errorf("expected 2 stops after sync, got %d", len(stops))
	}

	// Sync again with different data — should replace, not append
	newRoutes := []lta.BusRoute{
		{ServiceNo: "10", Direction: 1, StopSequence: 1, BusStopCode: "B1", Distance: 0},
	}
	if err := s.SyncRoutes(newRoutes); err != nil {
		t.Fatalf("second SyncRoutes failed: %v", err)
	}

	stops, _ = s.GetStopsByService("10")
	if len(stops) != 1 {
		t.Errorf("expected 1 stop after re-sync, got %d", len(stops))
	}
	if stops[0].StopCode != "B1" {
		t.Errorf("expected B1 after re-sync, got %s", stops[0].StopCode)
	}
}

type stop struct {
	code     string
	lat, lng float64
}
