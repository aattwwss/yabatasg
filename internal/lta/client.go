package lta

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const defaultHost = "https://datamall2.mytransport.sg"

type Client struct {
	apiKey string
	host   string
	cache  map[string]*CacheEntry
	mu     sync.RWMutex
}

type CacheEntry struct {
	Data      *BusArrival
	ExpiresAt time.Time
}

func New(apiKey, host string) *Client {
	if host == "" {
		host = defaultHost
	}
	return &Client{
		apiKey: apiKey,
		host:   host,
		cache:  make(map[string]*CacheEntry),
	}
}

func (c *Client) GetBusArrival(ctx context.Context, busStopCode, serviceNumber string) (*BusArrival, error) {
	cacheKey := fmt.Sprintf("%s-%s", busStopCode, serviceNumber)
	c.mu.RLock()
	if entry, found := c.cache[cacheKey]; found && time.Now().Before(entry.ExpiresAt) {
		c.mu.RUnlock()
		return entry.Data, nil
	}
	c.mu.RUnlock()

	url := c.host + "/ltaodataservice/v3/BusArrival"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
	if err := json.Unmarshal(body, &busArrival); err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[cacheKey] = &CacheEntry{
		Data:      &busArrival,
		ExpiresAt: time.Now().Add(10 * time.Second),
	}
	c.mu.Unlock()

	return &busArrival, nil
}

func (c *Client) GetBusStops(ctx context.Context, skip int) (*Response[BusStop], error) {
	url := c.host + "/ltaodataservice/BusStops"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
	if err := json.Unmarshal(body, &busStops); err != nil {
		return nil, err
	}

	return &busStops, nil
}
