package database

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testDatabase string
	testPassword string
	testUsername string
	testPort     string
	testHost     string
	testSchema   string
)

func mustStartPostgresContainer() (func(context.Context) error, error) {
	testDatabase = "database"
	testPassword = "password"
	testUsername = "user"
	testSchema = "public"

	container, err := postgres.Run(
		context.Background(),
		"postgres:latest",
		postgres.WithDatabase(testDatabase),
		postgres.WithUsername(testUsername),
		postgres.WithPassword(testPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		return nil, err
	}

	// Get host and port
	testHost, err = container.Host(context.Background())
	if err != nil {
		return container.Terminate, err
	}

	mappedPort, err := container.MappedPort(context.Background(), "5432/tcp")
	if err != nil {
		return container.Terminate, err
	}
	testPort = mappedPort.Port()

	return container.Terminate, nil
}

func TestMain(m *testing.M) {
	teardown, err := mustStartPostgresContainer()
	if err != nil {
		log.Fatalf("could not start postgres container: %v", err)
	}

	m.Run()

	if teardown != nil && teardown(context.Background()) != nil {
		log.Fatalf("could not teardown postgres container: %v", err)
	}
}

func TestNew(t *testing.T) {
	srv := New(testDatabase, testPassword, testUsername, testPort, testHost, testSchema)
	if srv == nil {
		t.Fatal("New() returned nil")
	}
}

func TestHealth(t *testing.T) {
	srv := New(testDatabase, testPassword, testUsername, testPort, testHost, testSchema)

	stats := srv.Health()

	if stats["status"] != "up" {
		t.Fatalf("expected status to be up, got %s", stats["status"])
	}

	if _, ok := stats["error"]; ok {
		t.Fatalf("expected error not to be present")
	}

	if stats["message"] != "It's healthy" {
		t.Fatalf("expected message to be 'It's healthy', got %s", stats["message"])
	}
}

func TestClose(t *testing.T) {
	srv := New(testDatabase, testPassword, testUsername, testPort, testHost, testSchema)

	if srv.Close() != nil {
		t.Fatalf("expected Close() to return nil")
	}
}
