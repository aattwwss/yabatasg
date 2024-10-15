package yabatasg

import "time"

type BusArrival struct {
	BusStopCode string
	Services    []Service
}

type NextBus struct {
	OriginCode       string
	DestinationCode  string
	EstimatedArrival time.Time
	Monitored        int
	Latitude         string
	Longitude        string
	VisitNumber      string
	Load             string
	Feature          string
	Type             string
}

type Service struct {
	ServiceNumber string
	Operator      string
	NextBuses     []NextBus
}
