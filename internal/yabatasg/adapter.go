package yabatasg

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/aattwwss/yabatasg/pkg/ltaapi"
)

type ltaClient interface {
	GetBusArrival(ctx context.Context, busStopCode string, serviceNumber string) (*ltaapi.BusArrival, error)
	GetBusRoutes(ctx context.Context, skip int) (*ltaapi.Response[ltaapi.BusRoute], error)
	GetBusStops(ctx context.Context, skip int) (*ltaapi.Response[ltaapi.BusStop], error)
	GetBusServices(ctx context.Context, skip int) (*ltaapi.Response[ltaapi.BusService], error)
}

type LTAClientAdapter struct {
	client ltaClient
}

func (lta *LTAClientAdapter) GetBusArrival(ctx context.Context, busStopCode string, serviceNumber string) (*BusArrival, error) {
	ba, err := lta.client.GetBusArrival(ctx, busStopCode, serviceNumber)
	if err != nil {
		return nil, err
	}

	services := []Service{}
	for _, serv := range ba.Services {
		service := Service{
			ServiceNumber: serv.ServiceNumber,
			Operator:      serv.Operator,
			NextBuses: []NextBus{
				nextBusMapper(serv.NextBus),
				nextBusMapper(serv.NextBus2),
				nextBusMapper(serv.NextBus3),
			},
		}
		services = append(services, service)
	}

	res := BusArrival{
		BusStopCode: ba.BusStopCode,
		Services:    services,
	}

	return &res, nil
}

func nextBusMapper(nb ltaapi.NextBus) NextBus {
	return NextBus{
		OriginCode:       nb.OriginCode,
		DestinationCode:  nb.DestinationCode,
		EstimatedArrival: nb.EstimatedArrival,
		Monitored:        nb.Monitored,
		Latitude:         nb.Latitude,
		Longitude:        nb.Longitude,
		VisitNumber:      nb.VisitNumber,
		Load:             nb.Load,
		Feature:          nb.Feature,
		Type:             nb.Type,
	}
}

func (lta *LTAClientAdapter) GetBusRoutes(ctx context.Context, skip int) ([]BusRoute, error) {
	brs, err := lta.client.GetBusRoutes(ctx, skip)
	if err != nil {
		return nil, err
	}

	res := []BusRoute{}
	for _, br := range brs.Value {
		busRoute := BusRoute{
			ServiceNumber:   br.ServiceNumber,
			Operator:        br.Operator,
			Direction:       br.Direction,
			StopSequence:    br.StopSequence,
			BusStopCode:     br.BusStopCode,
			Distance:        br.Distance,
			WeekDayFirstBus: parseTime(br.WeekDayFirstBus),
			WeekDayLastBus:  parseTime(br.WeekDayLastBus),
			SATFirstBus:     parseTime(br.SATFirstBus),
			SATLastBus:      parseTime(br.SATLastBus),
			SUNFirstBus:     parseTime(br.SUNFirstBus),
			SUNLastBus:      parseTime(br.SUNLastBus),
		}
		res = append(res, busRoute)
	}
	return res, nil
}

// parse hhmm into time type, return zero value time if there's error
func parseTime(timeString string) time.Time {
	t, err := time.Parse("1504", timeString)
	if err != nil {
		return time.Time{}

	}
	return t
}

func (lta *LTAClientAdapter) GetBusServices(ctx context.Context, skip int) ([]BusService, error) {
	bs, err := lta.client.GetBusServices(ctx, skip)
	if err != nil {
		return nil, err
	}

	res := []BusService{}
	for _, b := range bs.Value {
		busService := BusService{
			ServiceNumber:   b.ServiceNumber,
			Operator:        b.Operator,
			Direction:       b.Direction,
			Category:        b.Category,
			OriginCode:      b.OriginCode,
			DestinationCode: b.DestinationCode,
			LoopDesc:        b.LoopDesc,
		}
		busService.AMPeakFreqMin, busService.AMPeakFreqMax = splitFreqIntoMinMax(b.AMPeakFreq)
		busService.AMOffpeakFreqMin, busService.AMOffpeakFreqMax = splitFreqIntoMinMax(b.AMOffpeakFreq)
		busService.PMPeakFreqMin, busService.PMPeakFreqMax = splitFreqIntoMinMax(b.PMPeakFreq)
		busService.PMOffpeakFreqMin, busService.PMOffpeakFreqMax = splitFreqIntoMinMax(b.PMOffpeakFreq)
		res = append(res, busService)

	}
	return res, nil

}

func splitFreqIntoMinMax(freq string) (int, int) {
	var min int
	var max int
	split := strings.Split(freq, "-")
	if len(split) == 2 {
		min, _ = strconv.Atoi(split[0])
		max, _ = strconv.Atoi(split[1])
	}
	return min, max
}

func (lta *LTAClientAdapter) GetBusStops(ctx context.Context, skip int) ([]BusStop, error) {
	bs, err := lta.client.GetBusStops(ctx, skip)
	if err != nil {
		return nil, err
	}

	res := []BusStop{}
	for _, b := range bs.Value {
		busStop := BusStop{
			BusStopCode: b.BusStopCode,
			RoadName:    b.RoadName,
			Description: b.Description,
			Latitude:    b.Latitude,
			Longitude:   b.Longitude,
		}
		res = append(res, busStop)

	}
	return res, nil
}
