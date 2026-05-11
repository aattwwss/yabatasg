package store

import (
	"database/sql"
	"math"
	"sort"
	"time"

	"github.com/aattwwss/yabatasg/internal/lta"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type StopWithDistance struct {
	Code        string  `json:"code"`
	RoadName    string  `json:"roadName"`
	Description string  `json:"description"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Distance    float64 `json:"distance"`
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS bus_stops (
			code        TEXT PRIMARY KEY,
			road_name   TEXT NOT NULL,
			description TEXT NOT NULL,
			latitude    REAL NOT NULL,
			longitude   REAL NOT NULL
		);
		CREATE TABLE IF NOT EXISTS meta (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
	`)
	if err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Sync(stops []lta.BusStop) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO bus_stops (code, road_name, description, latitude, longitude) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, st := range stops {
		_, err = stmt.Exec(st.BusStopCode, st.RoadName, st.Description, st.Latitude, st.Longitude)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(`INSERT OR REPLACE INTO meta (key, value) VALUES ('last_synced', ?)`, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) LastSynced() (time.Time, error) {
	var val string
	err := s.db.QueryRow(`SELECT value FROM meta WHERE key = 'last_synced'`).Scan(&val)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, val)
}

func (s *Store) Nearby(lat, lng float64, limit int) ([]StopWithDistance, error) {
	// bounding box: ~3 km radius (~0.027 degrees)
	dlat := 0.027
	dlng := 0.027 / math.Cos(lat*math.Pi/180)

	rows, err := s.db.Query(`
		SELECT code, road_name, description, latitude, longitude
		FROM bus_stops
		WHERE latitude  BETWEEN ? AND ?
		  AND longitude BETWEEN ? AND ?
	`, lat-dlat, lat+dlat, lng-dlng, lng+dlng)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []StopWithDistance
	for rows.Next() {
		var swd StopWithDistance
		if err := rows.Scan(&swd.Code, &swd.RoadName, &swd.Description, &swd.Latitude, &swd.Longitude); err != nil {
			return nil, err
		}
		swd.Distance = haversine(lat, lng, swd.Latitude, swd.Longitude)
		results = append(results, swd)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}

	return results, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000 // Earth radius in meters
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
