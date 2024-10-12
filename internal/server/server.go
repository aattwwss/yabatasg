package server

import (
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
	syncAPIDuration, _ := strconv.Atoi(os.Getenv("SYNC_INTERVAL_MINUTES"))

	scheduler := scheduler.NewScheduler()
	scheduler.AddTask("test", time.Duration(syncAPIDuration)*time.Minute, func() {
		time.Sleep(time.Duration(10) * time.Second)
		slog.Info(time.Now().String())
	})

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
