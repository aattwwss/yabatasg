package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/aattwwss/yabatasg/internal/database"
	"github.com/aattwwss/yabatasg/internal/scheduler"
	"github.com/aattwwss/yabatasg/internal/yabatasg"
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"

	"github.com/aattwwss/yabatasg/pkg/ltaapi"
)

type Server struct {
	port int

	db           database.Service
	scheduler    *scheduler.Scheduler
	crawler      *yabatasg.Crawler
	ltaAPICleint ltaapi.Client
}

func NewServer() *http.Server {
	if err := godotenv.Load(); err != nil {
		slog.Error("Error loading .env file")
		os.Exit(1)
	}

	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		slog.Error("Parse env error", "error", err)
		os.Exit(1)
	}

	scheduler := scheduler.NewScheduler()
	db := database.New(cfg.DBDatabase, cfg.DBPassword, cfg.DBUsername, cfg.DBPort, cfg.DBHost, cfg.DBSchema)
	ltaAPIClient := ltaapi.New(cfg.LTAAccessKey, cfg.LTAAPIHost)
	ltaAPIlientAdapter := yabatasg.NewLTAClientAdapter(&ltaAPIClient)
	crawler := yabatasg.NewCrawler(ltaAPIlientAdapter, db)

	newServer := &Server{
		port:         cfg.Port,
		db:           db,
		scheduler:    scheduler,
		crawler:      crawler,
		ltaAPICleint: ltaAPIClient,
	}

	mux := newServer.RegisterRoutes()
	newServer.initTasksToScheduler(cfg, scheduler, crawler)
	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", newServer.port),
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}

func (s *Server) initTasksToScheduler(cfg Config, scheduler *scheduler.Scheduler, crawler *yabatasg.Crawler) {
	scheduler.AddTask("lta-crawler", time.Duration(cfg.SyncIntervalMinutes)*time.Minute, func(ctx context.Context) {
		_ = crawler.CrawlBusStops(ctx)
		_ = crawler.CrawlBusServices(ctx)
		_ = crawler.CrawlBusRoutes(ctx)
	})
}
