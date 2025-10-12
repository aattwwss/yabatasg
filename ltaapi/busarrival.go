package ltaapi

import (
	"encoding/json"
	"time"
)

// SafeTime is a wrapper around time.Time that safely unmarshals empty strings as zero time.
type SafeTime struct {
	time.Time
}

// UnmarshalJSON implements json.Unmarshaler.
// It handles empty strings by setting time to zero value.
func (st *SafeTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	if s == "" {
		st.Time = time.Time{} // zero value
		return nil
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	st.Time = t
	return nil
}

type BusArrival struct {
	BusStopCode string    `json:"BusStopCode"`
	Services    []Service `json:"Services"`
}

type NextBus struct {
	OriginCode       string   `json:"OriginCode"`
	DestinationCode  string   `json:"DestinationCode"`
	EstimatedArrival SafeTime `json:"EstimatedArrival"`
	Monitored        int      `json:"Monitored"`
	Latitude         string   `json:"Latitude"`
	Longitude        string   `json:"Longitude"`
	VisitNumber      string   `json:"VisitNumber"`
	Load             string   `json:"Load"`
	Feature          string   `json:"Feature"`
	Type             string   `json:"Type"`
}

type Service struct {
	ServiceNumber string  `json:"ServiceNo"`
	Operator      string  `json:"Operator"`
	NextBus       NextBus `json:"NextBus"`
	NextBus2      NextBus `json:"NextBus2"`
	NextBus3      NextBus `json:"NextBus3"`
}
