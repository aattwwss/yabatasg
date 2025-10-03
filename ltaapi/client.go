package ltaapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

type Client struct {
	apiKey string
	host   string
}

const defaultHost = "https://datamall2.mytransport.sg"

func New(apiKey string, host string) Client {
	if host == "" {
		host = defaultHost
	}
	return Client{
		apiKey: apiKey,
		host:   host,
	}
}

func (c *Client) GetBusArrival(ctx context.Context, busStopCode string, serviceNumber string) (*BusArrival, error) {
	url := c.host + "/ltaodataservice/v3/BusArrival"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("BusStopCode", busStopCode)
	q.Add("ServiceNo", serviceNumber)
	req.URL.RawQuery = q.Encode()

	req.Header.Add("AccountKey", c.apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var busArrival BusArrival
	err = json.Unmarshal(body, &busArrival)
	if err != nil {
		return nil, err
	}

	return &busArrival, nil
}

func (c *Client) GetBusRoutes(ctx context.Context, skip int) (*Response[BusRoute], error) {
	url := c.host + "/ltaodataservice/BusRoutes"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("$skip", strconv.Itoa(skip))
	req.URL.RawQuery = q.Encode()

	req.Header.Add("AccountKey", c.apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var busRoutes Response[BusRoute]
	err = json.Unmarshal(body, &busRoutes)
	if err != nil {
		return nil, err
	}

	return &busRoutes, nil
}

func (c *Client) GetBusStops(ctx context.Context, skip int) (*Response[BusStop], error) {
	url := c.host + "/ltaodataservice/BusStops"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("$skip", strconv.Itoa(skip))
	req.URL.RawQuery = q.Encode()

	req.Header.Add("AccountKey", c.apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var busStops Response[BusStop]
	err = json.Unmarshal(body, &busStops)
	if err != nil {
		return nil, err
	}

	return &busStops, nil
}

func (c *Client) GetBusServices(ctx context.Context, skip int) (*Response[BusService], error) {
	url := c.host + "/ltaodataservice/BusServices"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("$skip", strconv.Itoa(skip))
	req.URL.RawQuery = q.Encode()

	req.Header.Add("AccountKey", c.apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var busServices Response[BusService]
	err = json.Unmarshal(body, &busServices)
	if err != nil {
		return nil, err
	}

	return &busServices, nil
}
