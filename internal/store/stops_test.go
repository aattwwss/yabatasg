package store

import (
	"math"
	"os"
	"testing"
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
	dbPath := "test_nearby.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
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
	dbPath := "test_limit.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
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
	dbPath := "test_synced.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
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

type stop struct {
	code      string
	lat, lng  float64
}
