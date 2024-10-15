package crawlers

import (
	"context"
)

type apiClient interface {
	GetBusArrival(ctx context.Context, busStopCode string, serviceNumber string)
	GetBusRoutes(ctx context.Context, skip int)
	GetBusStops(ctx context.Context, skip int)
	GetBusServices(ctx context.Context, skip int)
}

type dbStore interface {
}

type LTABusServiceCrawler struct {
	client apiClient
	store  dbStore
}
