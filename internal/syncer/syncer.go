package syncer

import (
	"context"
	"log/slog"
	"time"

	"github.com/aattwwss/yabatasg/internal/lta"
	"github.com/aattwwss/yabatasg/internal/store"
)

type LTAClient interface {
	GetBusStops(ctx context.Context, skip int) (*lta.Response[lta.BusStop], error)
	GetBusRoutes(ctx context.Context, skip int) (*lta.Response[lta.BusRoute], error)
	GetBusArrival(ctx context.Context, busStopCode, serviceNumber string) (*lta.BusArrival, error)
}

type Syncer struct {
	store  *store.Store
	client LTAClient
}

func New(s *store.Store, c LTAClient) *Syncer {
	return &Syncer{store: s, client: c}
}

func (sy *Syncer) Run(ctx context.Context) {
	interval := 7 * 24 * time.Hour

	last, err := sy.store.LastSynced()
	if err != nil {
		slog.Error("Failed to check last synced time", "error", err)
	}
	if last.IsZero() {
		slog.Info("No bus stops data found, starting initial sync")
		sy.SyncNow(ctx)
	} else if time.Since(last) > 7*24*time.Hour {
		slog.Info("Last sync was more than 7 days ago, scheduling next sync in 1 day")
		interval = 24 * time.Hour
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			slog.Info("Starting scheduled bus stops sync")
			sy.SyncNow(ctx)
			if interval != 7*24*time.Hour {
				interval = 7 * 24 * time.Hour
				ticker.Reset(interval)
			}
		}
	}
}

func (sy *Syncer) SyncNow(ctx context.Context) error {
	slog.Info("Syncing bus stops from LTA")
	var all []lta.BusStop

	for skip := 0; ; skip += 500 {
		res, err := sy.client.GetBusStops(ctx, skip)
		if err != nil {
			slog.Error("Failed to fetch bus stops", "skip", skip, "error", err)
			return err
		}

		all = append(all, res.Value...)

		if len(res.Value) < 500 {
			break
		}
	}

	if err := sy.store.Sync(all); err != nil {
		slog.Error("Failed to sync bus stops to store", "error", err)
		return err
	}

	slog.Info("Bus stops synced", "count", len(all))

	slog.Info("Syncing bus routes from LTA")
	var allRoutes []lta.BusRoute
	for skip := 0; ; skip += 500 {
		res, err := sy.client.GetBusRoutes(ctx, skip)
		if err != nil {
			slog.Error("Failed to fetch bus routes", "skip", skip, "error", err)
			return err
		}
		allRoutes = append(allRoutes, res.Value...)
		if len(res.Value) < 500 {
			break
		}
	}

	if err := sy.store.SyncRoutes(allRoutes); err != nil {
		slog.Error("Failed to sync bus routes to store", "error", err)
		return err
	}

	slog.Info("Bus routes synced", "count", len(allRoutes))

	if err := sy.store.SeedServiceOperators(); err != nil {
		slog.Error("Failed to seed service operators", "error", err)
	}

	sy.syncOperators(ctx)

	return nil
}

func (sy *Syncer) queryStopsForOperators(ctx context.Context, stops map[string][]string) int {
	var synced int
	for stopCode := range stops {
		arrivals, err := sy.client.GetBusArrival(ctx, stopCode, "")
		if err != nil {
			slog.Warn("Failed to fetch arrivals for operator sync", "stopCode", stopCode, "error", err)
			continue
		}
		for _, svc := range arrivals.Services {
			if svc.Operator != "" {
				if err := sy.store.UpsertServiceOperator(svc.ServiceNumber, svc.Operator); err != nil {
					slog.Warn("Failed to upsert operator", "serviceNo", svc.ServiceNumber, "error", err)
				}
			}
		}
		synced++
	}
	return synced
}

func (sy *Syncer) syncOperators(ctx context.Context) {
	refs, err := sy.store.DistinctServiceStops()
	if err != nil {
		slog.Error("Failed to get distinct services for operator sync", "error", err)
		return
	}

	// Group by stop code to minimize API calls.
	byStop := make(map[string][]string)
	queriedStops := make(map[string]bool)
	for _, r := range refs {
		byStop[r.StopCode] = append(byStop[r.StopCode], r.ServiceNo)
		queriedStops[r.StopCode] = true
	}

	slog.Info("Syncing bus operators", "stops", len(byStop), "services", len(refs))
	synced := sy.queryStopsForOperators(ctx, byStop)
	slog.Info("Bus operators first pass", "stops_queried", synced)

	// Second pass: services still missing operators, try different stops.
	missing, err := sy.store.MissingOperatorServices()
	if err != nil {
		slog.Error("Failed to get missing operator services", "error", err)
		return
	}
	if len(missing) == 0 {
		return
	}

	// Second pass: use the other direction's first stop (direction 2, stop 1).
	altRefs, err := sy.store.AlternateServiceStops()
	if err != nil {
		slog.Error("Failed to get alternate stops", "error", err)
		return
	}
	byStop = make(map[string][]string)
	for _, r := range altRefs {
		if !queriedStops[r.StopCode] {
			byStop[r.StopCode] = append(byStop[r.StopCode], r.ServiceNo)
		}
	}

	if len(byStop) > 0 {
		slog.Info("Syncing bus operators (retry)", "stops", len(byStop), "services", len(missing))
		synced = sy.queryStopsForOperators(ctx, byStop)
		slog.Info("Bus operators second pass", "stops_queried", synced)
	}
}
