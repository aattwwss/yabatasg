package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/aattwwss/yabatasg/internal/database"
	"github.com/aattwwss/yabatasg/internal/scheduler"
	"github.com/aattwwss/yabatasg/pkg/ltaapi"
)

type Server struct {
	port int

	db           database.Service
	ltaAPIClient ltaapi.Client
	scheduler    *scheduler.Scheduler
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	accessKey := os.Getenv("LTA_ACCESS_KEY")

	scheduler := scheduler.NewScheduler()
	initTasksToScheduler(scheduler)

	newServer := &Server{
		port: port,

		db:           database.New(),
		ltaAPIClient: ltaapi.New(accessKey, ""),
		scheduler:    scheduler,
	}

	mux := newServer.RegisterRoutes()
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

func initTasksToScheduler(scheduler *scheduler.Scheduler) {
	syncAPIDuration, _ := strconv.Atoi(os.Getenv("SYNC_INTERVAL_MINUTES"))
	scheduler.AddTask("test", time.Duration(syncAPIDuration)*time.Minute, func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Counting stopped")
				return
			default:
				slog.Info(time.Now().String())
				time.Sleep(time.Second) // Wait for 1 second before printing the next number
			}
		}
	})
}
