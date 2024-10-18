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
	saveBuService(ctx context.Context, services []BusService) error
	saveBusStops(ctx context.Context, services []BusStop) error
	saveBusRoutes(ctx context.Context, services []BusRoute) error
}

type APICrawler struct {
	client apiClient
	store  dbStore
}

const maxBatchSize = 500

// CrawlBusServices will paginate through the api and save the results
func (a *APICrawler) CrawlBusServices(ctx context.Context, offset int) error {
	for {
		services, err := a.client.GetBusServices(ctx, offset)
		if err != nil {
			return err
		}

		if len(services) == 0 {
			break
		}

		a.store.saveBuService(ctx, services)
		offset += maxBatchSize
	}

	return nil
}

// CrawlBusStops will paginate through the api and save the results
func (a *APICrawler) CrawlBusStops(ctx context.Context, offset int) error {
	for {
		busStops, err := a.client.GetBusStops(ctx, offset)
		if err != nil {
			return err
		}

		if len(busStops) == 0 {
			break
		}

		a.store.saveBusStops(ctx, busStops)
		offset += maxBatchSize
	}

	return nil
}

// CrawlBusRoutes will paginate through the api and save the results
func (a *APICrawler) CrawlBusRoutes(ctx context.Context, offset int) error {
	for {
		services, err := a.client.GetBusRoutes(ctx, offset)
		if err != nil {
			return err
		}

		if len(services) == 0 {
			break
		}

		a.store.saveBusRoutes(ctx, services)
		offset += maxBatchSize
	}

	return nil
}
