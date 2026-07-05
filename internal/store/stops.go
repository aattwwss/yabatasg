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
		CREATE TABLE IF NOT EXISTS bus_routes (
			service_no   TEXT NOT NULL,
			direction    INTEGER NOT NULL,
			stop_sequence INTEGER NOT NULL,
			bus_stop_code TEXT NOT NULL,
			distance     REAL NOT NULL,
			PRIMARY KEY (service_no, direction, stop_sequence)
		);
		CREATE INDEX IF NOT EXISTS idx_bus_routes_service ON bus_routes(service_no);
		CREATE TABLE IF NOT EXISTS bus_services (
			service_no TEXT PRIMARY KEY,
			operator   TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS users (
			id         TEXT PRIMARY KEY,
			phrase     TEXT UNIQUE NOT NULL,
			token      TEXT UNIQUE NOT NULL,
			config     TEXT NOT NULL DEFAULT '[]',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_users_phrase ON users(phrase);
		CREATE INDEX IF NOT EXISTS idx_users_token  ON users(token);
	`)
	if err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

type Stop struct {
	Code        string  `json:"code"`
	RoadName    string  `json:"roadName"`
	Description string  `json:"description"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

func (s *Store) GetStop(code string) (*Stop, error) {
	var stop Stop
	err := s.db.QueryRow(
		`SELECT code, road_name, description, latitude, longitude FROM bus_stops WHERE code = ?`, code,
	).Scan(&stop.Code, &stop.RoadName, &stop.Description, &stop.Latitude, &stop.Longitude)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &stop, nil
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

func (s *Store) GetAllStopCodes() ([]string, error) {
	rows, err := s.db.Query(`SELECT code FROM bus_stops ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

type ServiceStop struct {
	StopCode    string `json:"stopCode"`
	RoadName    string `json:"roadName"`
	Description string `json:"description"`
	Direction   int    `json:"direction"`
	Sequence    int    `json:"sequence"`
}

type ServiceSearchResult struct {
	ServiceNo string `json:"serviceNo"`
	Operator  string `json:"operator"`
}

func (s *Store) SearchServices(query string) ([]ServiceSearchResult, error) {
	rows, err := s.db.Query(
		`SELECT service_no, operator FROM bus_services WHERE service_no LIKE ? ORDER BY service_no`,
		query+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ServiceSearchResult
	for rows.Next() {
		var r ServiceSearchResult
		if err := rows.Scan(&r.ServiceNo, &r.Operator); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) GetServiceOperator(serviceNo string) (string, error) {
	var operator string
	err := s.db.QueryRow(`SELECT operator FROM bus_services WHERE service_no = ?`, serviceNo).Scan(&operator)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return operator, nil
}

func (s *Store) UpsertServiceOperator(serviceNo, operator string) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO bus_services (service_no, operator) VALUES (?, ?)`,
		serviceNo, operator,
	)
	return err
}

func (s *Store) SeedServiceOperators() error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO bus_services (service_no, operator)
		 SELECT DISTINCT service_no, '' FROM bus_routes`,
	)
	return err
}

type ServiceStopRef struct {
	ServiceNo string
	StopCode  string
}

func (s *Store) DistinctServiceStops() ([]ServiceStopRef, error) {
	rows, err := s.db.Query(
		`SELECT r.service_no, r.bus_stop_code FROM bus_routes r
		 WHERE r.stop_sequence = 1 AND r.direction = 1`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []ServiceStopRef
	for rows.Next() {
		var r ServiceStopRef
		if err := rows.Scan(&r.ServiceNo, &r.StopCode); err != nil {
			return nil, err
		}
		refs = append(refs, r)
	}
	return refs, rows.Err()
}

func (s *Store) MissingOperatorServices() ([]string, error) {
	rows, err := s.db.Query(`SELECT service_no FROM bus_services WHERE operator = ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var svcs []string
	for rows.Next() {
		var no string
		if err := rows.Scan(&no); err != nil {
			return nil, err
		}
		svcs = append(svcs, no)
	}
	return svcs, rows.Err()
}

func (s *Store) AlternateServiceStops() ([]ServiceStopRef, error) {
	rows, err := s.db.Query(
		`SELECT r.service_no, r.bus_stop_code FROM bus_routes r
		 JOIN bus_services bs ON r.service_no = bs.service_no
		 WHERE bs.operator = '' AND r.stop_sequence = 1 AND r.direction = 2`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []ServiceStopRef
	for rows.Next() {
		var r ServiceStopRef
		if err := rows.Scan(&r.ServiceNo, &r.StopCode); err != nil {
			return nil, err
		}
		refs = append(refs, r)
	}
	return refs, rows.Err()
}

func (s *Store) GetStopsByService(serviceNo string) ([]ServiceStop, error) {
	rows, err := s.db.Query(`
		SELECT r.bus_stop_code, COALESCE(s.road_name, ''), COALESCE(s.description, ''), r.direction, r.stop_sequence
		FROM bus_routes r
		LEFT JOIN bus_stops s ON r.bus_stop_code = s.code
		WHERE r.service_no = ?
		ORDER BY r.direction, r.stop_sequence
	`, serviceNo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ServiceStop
	for rows.Next() {
		var st ServiceStop
		if err := rows.Scan(&st.StopCode, &st.RoadName, &st.Description, &st.Direction, &st.Sequence); err != nil {
			return nil, err
		}
		results = append(results, st)
	}
	return results, rows.Err()
}

func (s *Store) SyncRoutes(routes []lta.BusRoute) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM bus_routes`); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT INTO bus_routes (service_no, direction, stop_sequence, bus_stop_code, distance) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range routes {
		if _, err := stmt.Exec(r.ServiceNo, r.Direction, r.StopSequence, r.BusStopCode, r.Distance); err != nil {
			return err
		}
	}

	return tx.Commit()
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
