package yabatasg

import (
	"context"
)

type apiClient interface {
	GetBusArrival(ctx context.Context, busStopCode string, serviceNumber string) (*BusArrival, error)
	GetBusRoutes(ctx context.Context, skip int) ([]BusRoute, error)
	GetBusStops(ctx context.Context, skip int) ([]BusStop, error)
	GetBusServices(ctx context.Context, skip int) ([]BusService, error)
}

type dbStore interface {
	SaveBuService(ctx context.Context, services []BusService) error
	SaveBusStops(ctx context.Context, stop []BusStop) error
	SaveBusRoutes(ctx context.Context, routes []BusRoute) error
}

type Crawler struct {
	client apiClient
	store  dbStore
}

func NewCrawler(apiClient apiClient, store dbStore) *Crawler {
	return &Crawler{apiClient, store}
}

const maxBatchSize = 500

// CrawlBusServices will paginate through the api and save the results
func (c *Crawler) CrawlBusServices(ctx context.Context) error {
	offset := 0
	for {
		services, err := c.client.GetBusServices(ctx, offset)
		if err != nil {
			return err
		}

		if len(services) == 0 {
			break
		}

		c.store.SaveBuService(ctx, services)
		offset += maxBatchSize
	}

	return nil
}

// CrawlBusStops will paginate through the api and save the results
func (c *Crawler) CrawlBusStops(ctx context.Context) error {
	offset := 0
	for {
		busStops, err := c.client.GetBusStops(ctx, offset)
		if err != nil {
			return err
		}

		if len(busStops) == 0 {
			break
		}

		c.store.SaveBusStops(ctx, busStops)
		offset += maxBatchSize
	}

	return nil
}

// CrawlBusRoutes will paginate through the api and save the results
func (c *Crawler) CrawlBusRoutes(ctx context.Context) error {
	offset := 0
	for {
		routes, err := c.client.GetBusRoutes(ctx, offset)
		if err != nil {
			return err
		}

		if len(routes) == 0 {
			break
		}

		c.store.SaveBusRoutes(ctx, routes)
		offset += maxBatchSize
	}

	return nil
}
