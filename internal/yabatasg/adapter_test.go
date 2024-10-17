package yabatasg

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aattwwss/yabatasg/pkg/ltaapi"
)

var nextBusTime, _ = time.Parse(time.RFC3339, "2024-10-12T14:23:00+08:00")
var nextBus2Time, _ = time.Parse(time.RFC3339, "2024-10-12T14:38:00+08:00")
var nextBus3Time, _ = time.Parse(time.RFC3339, "2024-10-12T14:53:00+08:00")

var mockArrivals = ltaapi.BusArrival{
	BusStopCode: "75009",
	Services: []ltaapi.Service{
		{
			ServiceNumber: "10",
			Operator:      "SBST",
			NextBus: ltaapi.NextBus{
				OriginCode:       "75009",
				DestinationCode:  "16009",
				EstimatedArrival: nextBusTime,
				Monitored:        0,
				Latitude:         "0.0",
				Longitude:        "0.0",
				VisitNumber:      "1",
				Load:             "SEA",
				Feature:          "WAB",
				Type:             "DD",
			},
			NextBus2: ltaapi.NextBus{
				OriginCode:       "75009",
				DestinationCode:  "16009",
				EstimatedArrival: nextBus2Time,
				Monitored:        0,
				Latitude:         "0.0",
				Longitude:        "0.0",
				VisitNumber:      "1",
				Load:             "SEA",
				Feature:          "WAB",
				Type:             "SD",
			},
			NextBus3: ltaapi.NextBus{
				OriginCode:       "75009",
				DestinationCode:  "16009",
				EstimatedArrival: nextBus3Time,
				Monitored:        0,
				Latitude:         "0.0",
				Longitude:        "0.0",
				VisitNumber:      "1",
				Load:             "SEA",
				Feature:          "WAB",
				Type:             "SD",
			},
		},
	},
}

var mockRoutes = []ltaapi.BusRoute{
	{
		ServiceNumber:   "10",
		Operator:        "SBST",
		Direction:       1,
		StopSequence:    1,
		BusStopCode:     "75009",
		Distance:        0,
		WeekDayFirstBus: "0510",
		WeekDayLastBus:  "2300",
		SATFirstBus:     "0500",
		SATLastBus:      "2300",
		SUNFirstBus:     "0500-",
		SUNLastBus:      "2300wrong",
	},
	{
		ServiceNumber:   "10",
		Operator:        "SBST",
		Direction:       1,
		StopSequence:    2,
		BusStopCode:     "76059",
		Distance:        0.6,
		WeekDayFirstBus: "0502",
		WeekDayLastBus:  "2302",
		SATFirstBus:     "0502",
		SATLastBus:      "2302",
		SUNFirstBus:     "0502",
		SUNLastBus:      "2302",
	},
}

var mockStops = []ltaapi.BusStop{
	{
		BusStopCode: "23211",
		RoadName:    "Benoi Sector",
		Description: "Mapletree Logistics Hub",
		Latitude:    1.31792061914698,
		Longitude:   103.6892047185557,
	},
	{
		BusStopCode: "23219",
		RoadName:    "Benoi Sector",
		Description: "Tru-Marine",
		Latitude:    1.31832727349422,
		Longitude:   103.68852528629336,
	},
}

var mockBusServices = []ltaapi.BusService{
	{
		ServiceNumber:   "13",
		Operator:        "SBST",
		Direction:       1,
		Category:        "TRUNK",
		OriginCode:      "94009",
		DestinationCode: "55509",
		AMPeakFreq:      "10-13",
		AMOffpeakFreq:   "09-13",
		PMPeakFreq:      "08-10",
		PMOffpeakFreq:   "11-18",
		LoopDesc:        "",
	},
	{
		ServiceNumber:   "13",
		Operator:        "SBST",
		Direction:       2,
		Category:        "TRUNK",
		OriginCode:      "94009",
		DestinationCode: "55509",
		AMPeakFreq:      "-",
		AMOffpeakFreq:   "-",
		PMPeakFreq:      "-",
		PMOffpeakFreq:   "-",
		LoopDesc:        "",
	},
}

type mockLTAClient struct {
}

func (m *mockLTAClient) GetBusArrival(ctx context.Context, busStopCode string, serviceNumber string) (*ltaapi.BusArrival, error) {
	return &mockArrivals, nil
}

func (m *mockLTAClient) GetBusRoutes(ctx context.Context, skip int) (*ltaapi.Response[ltaapi.BusRoute], error) {
	return &ltaapi.Response[ltaapi.BusRoute]{
		Value: mockRoutes,
	}, nil
}

func (m *mockLTAClient) GetBusStops(ctx context.Context, skip int) (*ltaapi.Response[ltaapi.BusStop], error) {
	return &ltaapi.Response[ltaapi.BusStop]{
		Value: mockStops,
	}, nil
}

func (m *mockLTAClient) GetBusServices(ctx context.Context, skip int) (*ltaapi.Response[ltaapi.BusService], error) {
	return &ltaapi.Response[ltaapi.BusService]{
		Value: mockBusServices,
	}, nil
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestGetBusArrival(t *testing.T) {
	adapter := LTAClientAdapter{
		client: &mockLTAClient{},
	}

	arrivals, _ := adapter.GetBusArrival(context.Background(), "any", "any")
	if len(arrivals.Services[0].NextBuses) != 3 {
		t.Errorf("want 3 next buses, got: %v", len(arrivals.Services[0].NextBuses))
	}

	get := arrivals.Services[0].NextBuses[0]
	want := NextBus{
		OriginCode:       "75009",
		DestinationCode:  "16009",
		EstimatedArrival: nextBusTime,
		Monitored:        0,
		Latitude:         "0.0",
		Longitude:        "0.0",
		VisitNumber:      "1",
		Load:             "SEA",
		Feature:          "WAB",
		Type:             "DD",
	}
	if want != get {
		t.Errorf("want: %v, got: %v", want, get)
	}
}

func TestGetBusRoutes(t *testing.T) {
	adapter := LTAClientAdapter{
		client: &mockLTAClient{},
	}

	routes, _ := adapter.GetBusRoutes(context.Background(), 123)
	if len(routes) != 2 {
		t.Errorf("want 2 routes, got: %v", len(routes))
	}

	get := routes[0]
	want := BusRoute{
		ServiceNumber:   "10",
		Operator:        "SBST",
		Direction:       1,
		StopSequence:    1,
		BusStopCode:     "75009",
		Distance:        0,
		WeekDayFirstBus: time.Date(0, 1, 1, 5, 10, 0, 0, time.UTC),
		WeekDayLastBus:  time.Date(0, 1, 1, 23, 0, 0, 0, time.UTC),
		SATFirstBus:     time.Date(0, 1, 1, 5, 0, 0, 0, time.UTC),
		SATLastBus:      time.Date(0, 1, 1, 23, 0, 0, 0, time.UTC),
		SUNFirstBus:     time.Time{}, // zero value if time cannot be parsed
		SUNLastBus:      time.Time{},
	}
	if want != get {
		t.Errorf("want: %v, got: %v", want, get)
	}
}

func TestGetBusStops(t *testing.T) {
	adapter := LTAClientAdapter{
		client: &mockLTAClient{},
	}

	stops, _ := adapter.GetBusStops(context.Background(), 123)
	if len(stops) != 2 {
		t.Errorf("want 2 routes, got: %v", len(stops))
	}

	get := stops[0]
	want := BusStop{
		BusStopCode: "23211",
		RoadName:    "Benoi Sector",
		Description: "Mapletree Logistics Hub",
		Latitude:    1.31792061914698,
		Longitude:   103.6892047185557,
	}
	if want != get {
		t.Errorf("want: %v, got: %v", want, get)
	}
}

func TestGetBusServices(t *testing.T) {
	adapter := LTAClientAdapter{
		client: &mockLTAClient{},
	}

	services, _ := adapter.GetBusServices(context.Background(), 123)
	if len(services) != 2 {
		t.Errorf("want 2 routes, got: %v", len(services))
	}
	tests := []struct {
		name string
		want BusService
		get  BusService
	}{
		{
			name: "standard",
			get:  services[0],
			want: BusService{
				ServiceNumber:    "13",
				Operator:         "SBST",
				Direction:        1,
				Category:         "TRUNK",
				OriginCode:       "94009",
				DestinationCode:  "55509",
				AMPeakFreqMin:    10,
				AMPeakFreqMax:    13,
				AMOffpeakFreqMin: 9,
				AMOffpeakFreqMax: 13,
				PMPeakFreqMin:    8,
				PMPeakFreqMax:    10,
				PMOffpeakFreqMin: 11,
				PMOffpeakFreqMax: 18,
				LoopDesc:         "",
			},
		},
		{
			name: "empty freq",
			get:  services[1],
			want: BusService{
				ServiceNumber:    "13",
				Operator:         "SBST",
				Direction:        2,
				Category:         "TRUNK",
				OriginCode:       "94009",
				DestinationCode:  "55509",
				AMPeakFreqMin:    0,
				AMPeakFreqMax:    0,
				AMOffpeakFreqMin: 0,
				AMOffpeakFreqMax: 0,
				PMPeakFreqMin:    0,
				PMPeakFreqMax:    0,
				PMOffpeakFreqMin: 0,
				PMOffpeakFreqMax: 0,
				LoopDesc:         "",
			},
		},
	}
	for _, tc := range tests {
		if tc.want != tc.get {
			t.Errorf("want: %v, got: %v", tc.want, tc.get)
		}
	}

}
