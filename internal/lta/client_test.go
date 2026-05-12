package lta

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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("AccountKey") != "test-api-key" {
			t.Errorf("Expected AccountKey header to be test-api-key")
		}
		if r.URL.Query().Get("BusStopCode") != "12345" {
			t.Errorf("Expected BusStopCode query parameter to be 12345")
		}
		if r.URL.Query().Get("ServiceNo") != "123" {
			t.Errorf("Expected ServiceNo query parameter to be 123")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(busArrivalResponse))
	}))
	defer server.Close()

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

	tests := []struct {
		name     string
		expected interface{}
		actual   interface{}
	}{
		{"BusStopCode", "75009", busArrival.BusStopCode},
		{"ServiceNo", "10", busArrival.Services[0].ServiceNumber},
		{"Operator", "SBST", busArrival.Services[0].Operator},
		{"NextBus OriginCode", "75009", busArrival.Services[0].NextBus.OriginCode},
		{"NextBus DestinationCode", "16009", busArrival.Services[0].NextBus.DestinationCode},
		{"NextBus EstimatedArrival", "2024-10-12T14:23:00+08:00", busArrival.Services[0].NextBus.EstimatedArrival.Format(time.RFC3339)},
		{"NextBus Latitude", "0.0", busArrival.Services[0].NextBus.Latitude},
		{"NextBus Longitude", "0.0", busArrival.Services[0].NextBus.Longitude},
		{"NextBus Load", "SEA", busArrival.Services[0].NextBus.Load},
		{"NextBus Feature", "WAB", busArrival.Services[0].NextBus.Feature},
		{"NextBus Type", "DD", busArrival.Services[0].NextBus.Type},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("mismatch: expected %v, got %v", tt.expected, tt.actual)
			}
		})
	}
}
