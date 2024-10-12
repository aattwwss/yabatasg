package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/aattwwss/yabatasg/internal/database"
	"github.com/aattwwss/yabatasg/pkg/ltaapi"
)

type Server struct {
	port int

	db           database.Service
	ltaAPIClient ltaapi.Client
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	accessKey := os.Getenv("LTA_ACCESS_KEY")
	NewServer := &Server{
		port: port,

		db:           database.New(),
		ltaAPIClient: ltaapi.New(accessKey, ""),
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
