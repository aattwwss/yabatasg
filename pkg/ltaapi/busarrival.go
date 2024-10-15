package ltaapi

import "time"

type BusArrival struct {
	BusStopCode string    `json:"BusStopCode"`
	Services    []Service `json:"Services"`
}

type NextBus struct {
	OriginCode       string    `json:"OriginCode"`
	DestinationCode  string    `json:"DestinationCode"`
	EstimatedArrival time.Time `json:"EstimatedArrival"`
	Monitored        int       `json:"Monitored"`
	Latitude         string    `json:"Latitude"`
	Longitude        string    `json:"Longitude"`
	VisitNumber      string    `json:"VisitNumber"`
	Load             string    `json:"Load"`
	Feature          string    `json:"Feature"`
	Type             string    `json:"Type"`
}

type Service struct {
	ServiceNo string  `json:"ServiceNo"`
	Operator  string  `json:"Operator"`
	NextBus   NextBus `json:"NextBus"`
	NextBus2  NextBus `json:"NextBus2"`
	NextBus3  NextBus `json:"NextBus3"`
}
