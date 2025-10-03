package ltaapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const busArrivalResponse = `{
    "odata.metadata": "https://datamall2.mytransport.sg/ltaodataservice/v3/BusArrival",
    "BusStopCode": "75009",
    "Services": [
        {
            "ServiceNo": "10",
            "Operator": "SBST",
            "NextBus": {
                "OriginCode": "75009",
                "DestinationCode": "16009",
                "EstimatedArrival": "2024-10-12T14:23:00+08:00",
                "Monitored": 0,
                "Latitude": "0.0",
                "Longitude": "0.0",
                "VisitNumber": "1",
                "Load": "SEA",
                "Feature": "WAB",
                "Type": "DD"
            },
            "NextBus2": {
                "OriginCode": "75009",
                "DestinationCode": "16009",
                "EstimatedArrival": "2024-10-12T14:38:00+08:00",
                "Monitored": 0,
                "Latitude": "0.0",
                "Longitude": "0.0",
                "VisitNumber": "1",
                "Load": "SEA",
                "Feature": "WAB",
                "Type": "SD"
            },
            "NextBus3": {
                "OriginCode": "75009",
                "DestinationCode": "16009",
                "EstimatedArrival": "2024-10-12T14:53:00+08:00",
                "Monitored": 0,
                "Latitude": "0.0",
                "Longitude": "0.0",
                "VisitNumber": "1",
                "Load": "SEA",
                "Feature": "WAB",
                "Type": "SD"
            }
        }
    ]
}`

func TestGetBusArrival(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request has the correct headers and query parameters
		if r.Header.Get("AccountKey") != "test-api-key" {
			t.Errorf("Expected AccountKey header to be test-api-key")
		}

		if r.URL.Query().Get("BusStopCode") != "12345" {
			t.Errorf("Expected BusStopCode query parameter to be 12345")
		}

		if r.URL.Query().Get("ServiceNo") != "123" {
			t.Errorf("Expected ServiceNo query parameter to be 123")
		}
		mockResponseJSON := busArrivalResponse

		// Prepare a mock response
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponseJSON))
	}))
	defer server.Close()

	// Create a new client and make the API call
	client := New("test-api-key", server.URL)
	busArrival, err := client.GetBusArrival(context.Background(), "12345", "123")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if busArrival == nil {
		t.Fatal("Expected non-nil BusArrival, but got nil")
	}

	if len(busArrival.Services) != 1 {
		t.Fatalf("Expected 1 timing, but got %d", len(busArrival.Services))
	}

	// Table-driven tests for field validation
	tests := []struct {
		name     string
		field    string
		expected interface{}
		actual   interface{}
	}{
		{"BusStopCode", "BusStopCode", "75009", busArrival.BusStopCode},
		{"ServiceNo", "Services[0].ServiceNo", "10", busArrival.Services[0].ServiceNumber},
		{"Operator", "Services[0].Operator", "SBST", busArrival.Services[0].Operator},
		{"NextBus OriginCode", "Services[0].NextBus.OriginCode", "75009", busArrival.Services[0].NextBus.OriginCode},
		{"NextBus DestinationCode", "Services[0].NextBus.DestinationCode", "16009", busArrival.Services[0].NextBus.DestinationCode},
		{"NextBus EstimatedArrival", "Services[0].NextBus.EstimatedArrival", "2024-10-12T14:23:00+08:00", busArrival.Services[0].NextBus.EstimatedArrival.Format(time.RFC3339)},
		{"NextBus Latitude", "Services[0].NextBus.Latitude", "0.0", busArrival.Services[0].NextBus.Latitude},
		{"NextBus Longitude", "Services[0].NextBus.Longitude", "0.0", busArrival.Services[0].NextBus.Longitude},
		{"NextBus Load", "Services[0].NextBus.Load", "SEA", busArrival.Services[0].NextBus.Load},
		{"NextBus Feature", "Services[0].NextBus.Feature", "WAB", busArrival.Services[0].NextBus.Feature},
		{"NextBus Type", "Services[0].NextBus.Type", "DD", busArrival.Services[0].NextBus.Type},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("Field %s mismatch: expected %v, got %v", tt.field, tt.expected, tt.actual)
			}
		})
	}

}

const busRoutesResponse = `{
  "odata.metadata": "http://datamall2.mytransport.sg/ltaodataservice/$metadataBusRoutes",
  "value": [
    {
      "ServiceNo": "10",
      "Operator": "SBST",
      "Direction": 1,
      "StopSequence": 1,
      "BusStopCode": "75009",
      "Distance": 0,
      "WD_FirstBus": "0500",
      "WD_LastBus": "2300",
      "SAT_FirstBus": "0500",
      "SAT_LastBus": "2300",
      "SUN_FirstBus": "0500",
      "SUN_LastBus": "2300"
    },
    {
      "ServiceNo": "10",
      "Operator": "SBST",
      "Direction": 1,
      "StopSequence": 2,
      "BusStopCode": "76059",
      "Distance": 0.6,
      "WD_FirstBus": "0502",
      "WD_LastBus": "2302",
      "SAT_FirstBus": "0502",
      "SAT_LastBus": "2302",
      "SUN_FirstBus": "0502",
      "SUN_LastBus": "2302"
    }
  ]
}`

func TestGetBusRoutes(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request has the correct headers and query parameters
		if r.Header.Get("AccountKey") != "test-api-key" {
			t.Errorf("Expected AccountKey header to be test-api-key")
		}

		if r.URL.Query().Get("$skip") != "123" {
			t.Errorf("Expected skip query parameter to be 123")
		}
		mockResponseJSON := busRoutesResponse

		// Prepare a mock response
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponseJSON))
	}))
	defer server.Close()

	// Create a new client and make the API call
	client := New("test-api-key", server.URL)
	busRoutes, err := client.GetBusRoutes(context.Background(), 123)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if busRoutes == nil {
		t.Fatal("Expected non-nil BusArrival, but got nil")
	}

	if len(busRoutes.Value) != 2 {
		t.Fatalf("Expected 2 values bus routes, but got %d", len(busRoutes.Value))
	}

	tests := []struct {
		name     string
		index    int
		field    string
		expected interface{}
		actual   interface{}
	}{
		{"ServiceNumber 1", 0, "ServiceNo", "10", busRoutes.Value[0].ServiceNumber},
		{"Operator 1", 0, "Operator", "SBST", busRoutes.Value[0].Operator},
		{"Direction 1", 0, "Direction", 1, busRoutes.Value[0].Direction},
		{"StopSequence 1", 0, "StopSequence", 1, busRoutes.Value[0].StopSequence},
		{"BusStopCode 1", 0, "BusStopCode", "75009", busRoutes.Value[0].BusStopCode},
		{"Distance 1", 0, "Distance", 0.0, busRoutes.Value[0].Distance},
		{"WeekDayFirstBus 1", 0, "WD_FirstBus", "0500", busRoutes.Value[0].WeekDayFirstBus},
		{"WeekDayLastBus 1", 0, "WD_LastBus", "2300", busRoutes.Value[0].WeekDayLastBus},
		{"SATFirstBus 1", 0, "SAT_FirstBus", "0500", busRoutes.Value[0].SATFirstBus},
		{"SATLastBus 1", 0, "SAT_LastBus", "2300", busRoutes.Value[0].SATLastBus},
		{"SUNFirstBus 1", 0, "SUN_FirstBus", "0500", busRoutes.Value[0].SUNFirstBus},
		{"SUNLastBus 1", 0, "SUN_LastBus", "2300", busRoutes.Value[0].SUNLastBus},

		{"ServiceNumber 2", 1, "ServiceNo", "10", busRoutes.Value[1].ServiceNumber},
		{"Operator 2", 1, "Operator", "SBST", busRoutes.Value[1].Operator},
		{"Direction 2", 1, "Direction", 1, busRoutes.Value[1].Direction},
		{"StopSequence 2", 1, "StopSequence", 2, busRoutes.Value[1].StopSequence},
		{"BusStopCode 2", 1, "BusStopCode", "76059", busRoutes.Value[1].BusStopCode},
		{"Distance 2", 1, "Distance", 0.6, busRoutes.Value[1].Distance}, // Note: This will fail due to type mismatch
		{"WeekDayFirstBus 2", 1, "WD_FirstBus", "0502", busRoutes.Value[1].WeekDayFirstBus},
		{"WeekDayLastBus 2", 1, "WD_LastBus", "2302", busRoutes.Value[1].WeekDayLastBus},
		{"SATFirstBus 2", 1, "SAT_FirstBus", "0502", busRoutes.Value[1].SATFirstBus},
		{"SATLastBus 2", 1, "SAT_LastBus", "2302", busRoutes.Value[1].SATLastBus},
		{"SUNFirstBus 2", 1, "SUN_FirstBus", "0502", busRoutes.Value[1].SUNFirstBus},
		{"SUNLastBus 2", 1, "SUN_LastBus", "2302", busRoutes.Value[1].SUNLastBus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("BusRoutes.Value[%d].%s mismatch: expected %v (%T), got %v (%T)",
					tt.index, tt.field, tt.expected, tt.expected, tt.actual, tt.actual)
			}
		})
	}

}

const busStopsResponse = `{
  "odata.metadata": "http://datamall2.mytransport.sg/ltaodataservice/$metadata#BusStops",
  "value": [
    {
      "BusStopCode": "23211",
      "RoadName": "Benoi Sector",
      "Description": "Mapletree Logistics Hub",
      "Latitude": 1.31792061914698,
      "Longitude": 103.6892047185557
    },
    {
      "BusStopCode": "23219",
      "RoadName": "Benoi Sector",
      "Description": "Tru-Marine",
      "Latitude": 1.31832727349422,
      "Longitude": 103.68852528629336
    }
  ]
}`

func TestGetBusStops(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request has the correct headers and query parameters
		if r.Header.Get("AccountKey") != "test-api-key" {
			t.Errorf("Expected AccountKey header to be test-api-key")
		}

		if r.URL.Query().Get("$skip") != "123" {
			t.Errorf("Expected skip query parameter to be 123")
		}
		mockResponseJSON := busStopsResponse

		// Prepare a mock response
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponseJSON))
	}))
	defer server.Close()

	// Create a new client and make the API call
	client := New("test-api-key", server.URL)
	busStops, err := client.GetBusStops(context.Background(), 123)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if busStops == nil {
		t.Fatal("Expected non-nil busStops, but got nil")
	}

	if len(busStops.Value) != 2 {
		t.Fatalf("Expected 2 values bus busStops, but got %d", len(busStops.Value))
	}

	tests := []struct {
		name     string
		index    int
		field    string
		expected interface{}
		actual   interface{}
	}{
		{"BusStopCode 1", 0, "BusStopCode", "23211", busStops.Value[0].BusStopCode},
		{"RoadName 1", 0, "RoadName", "Benoi Sector", busStops.Value[0].RoadName},
		{"Description 1", 0, "Description", "Mapletree Logistics Hub", busStops.Value[0].Description},
		{"Latitude 1", 0, "Latitude", 1.31792061914698, busStops.Value[0].Latitude},
		{"Longitude 1", 0, "Longitude", 103.6892047185557, busStops.Value[0].Longitude},

		{"BusStopCode 2", 1, "BusStopCode", "23219", busStops.Value[1].BusStopCode},
		{"RoadName 2", 1, "RoadName", "Benoi Sector", busStops.Value[1].RoadName},
		{"Description 2", 1, "Description", "Tru-Marine", busStops.Value[1].Description},
		{"Latitude 2", 1, "Latitude", 1.31832727349422, busStops.Value[1].Latitude},
		{"Longitude 2", 1, "Longitude", 103.68852528629336, busStops.Value[1].Longitude},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("BusStop[%d].%s mismatch: expected %v (%T), got %v (%T)",
					tt.index, tt.field, tt.expected, tt.expected, tt.actual, tt.actual)
			}
		})
	}
}

const busServicesResponse = `{
		"odata.metadata": "http://datamall2.mytransport.sg/ltaodataservice/$metadata#BusServices",
		"value": [
			{
				"ServiceNo": "13",
				"Operator": "SBST",
				"Direction": 2,
				"Category": "TRUNK",
				"OriginCode": "94009",
				"DestinationCode": "55509",
				"AM_Peak_Freq": "10-13",
				"AM_Offpeak_Freq": "09-13",
				"PM_Peak_Freq": "08-10",
				"PM_Offpeak_Freq": "11-18",
				"LoopDesc": ""
			}
		]
	}`

func TestGetBusServices(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request has the correct headers and query parameters
		if r.Header.Get("AccountKey") != "test-api-key" {
			t.Errorf("Expected AccountKey header to be test-api-key")
		}

		if r.URL.Query().Get("$skip") != "123" {
			t.Errorf("Expected skip query parameter to be 123")
		}
		mockResponseJSON := busServicesResponse

		// Prepare a mock response
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponseJSON))
	}))
	defer server.Close()

	// Create a new client and make the API call
	client := New("test-api-key", server.URL)
	busServices, err := client.GetBusServices(context.Background(), 123)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if busServices == nil {
		t.Fatal("Expected non-nil busServices, but got nil")
	}

	if len(busServices.Value) != 1 {
		t.Fatalf("Expected 1 values bus service, but got %d", len(busServices.Value))
	}
	tests := []struct {
		name     string
		field    string
		expected interface{}
		actual   interface{}
	}{
		{"ServiceNumber", "ServiceNo", "13", busServices.Value[0].ServiceNumber},
		{"Operator", "Operator", "SBST", busServices.Value[0].Operator},
		{"Direction", "Direction", 2, busServices.Value[0].Direction},
		{"Category", "Category", "TRUNK", busServices.Value[0].Category},
		{"OriginCode", "OriginCode", "94009", busServices.Value[0].OriginCode},
		{"DestinationCode", "DestinationCode", "55509", busServices.Value[0].DestinationCode},
		{"AMPeakFreq", "AM_Peak_Freq", "10-13", busServices.Value[0].AMPeakFreq},
		{"AMOffpeakFreq", "AM_Offpeak_Freq", "09-13", busServices.Value[0].AMOffpeakFreq},
		{"PMPeakFreq", "PM_Peak_Freq", "08-10", busServices.Value[0].PMPeakFreq},
		{"PMOffpeakFreq", "PM_Offpeak_Freq", "11-18", busServices.Value[0].PMOffpeakFreq},
		{"LoopDesc", "LoopDesc", "", busServices.Value[0].LoopDesc},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("BusService.%s mismatch: expected %v (%T), got %v (%T)",
					tt.field, tt.expected, tt.expected, tt.actual, tt.actual)
			}
		})
	}
}
