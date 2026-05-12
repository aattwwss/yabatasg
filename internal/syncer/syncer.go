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
}

type Syncer struct {
	store  *store.Store
	client LTAClient
}

func New(s *store.Store, c LTAClient) *Syncer {
	return &Syncer{store: s, client: c}
}

func (sy *Syncer) Run(ctx context.Context) {
	ticker := time.NewTicker(7 * 24 * time.Hour)
	defer ticker.Stop()

	last, err := sy.store.LastSynced()
	if err != nil {
		slog.Error("Failed to check last synced time", "error", err)
	}
	if last.IsZero() {
		slog.Info("No bus stops data found, starting initial sync")
		sy.SyncNow()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			slog.Info("Starting scheduled bus stops sync")
			sy.SyncNow()
		}
	}
}

func (sy *Syncer) SyncNow() error {
	slog.Info("Syncing bus stops from LTA")
	var all []lta.BusStop

	for skip := 0; ; skip += 500 {
		res, err := sy.client.GetBusStops(context.Background(), skip)
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
	return nil
}
