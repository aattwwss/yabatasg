package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type taskRunner struct {
	running bool
	mutex   sync.Mutex
	cancel  context.CancelFunc
	execute func(ctx context.Context)
}

func (t *taskRunner) start() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.running {
		slog.Info("asdasdf", "running", t.running)
		return fmt.Errorf("task is already running")
	}

	t.running = true
	ctx, cancelFunc := context.WithCancel(context.Background())
	t.cancel = cancelFunc

	go t.execute(ctx)

	return nil
}

func (t *taskRunner) stop() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.running {
		return fmt.Errorf("task is already stopped")
	}

	t.cancel()
	t.running = false
	return nil
}

type Task struct {
	ID       string
	Interval time.Duration
	runner   taskRunner
	stop     chan bool
	enabled  bool
	mutex    sync.Mutex
}

type Scheduler struct {
	mutex sync.Mutex
	tasks map[string]*Task
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks: make(map[string]*Task),
	}
}

func (s *Scheduler) AddTask(id string, interval time.Duration, t func(ctx context.Context)) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	task := &Task{
		ID:       id,
		Interval: interval,
		stop:     make(chan bool),
		runner: taskRunner{
			running: false,
			execute: t,
		},
	}
	s.tasks[id] = task
}

func (s *Scheduler) StopTask(ctx context.Context, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	task, exists := s.tasks[id]

	if !exists {
		return fmt.Errorf("task with ID %s not found", id)
	}

	if !task.runner.running {
		return fmt.Errorf("task with ID %s is already stopped", id)
	}

	err := task.runner.stop()
	if err != nil {
		return fmt.Errorf("task with ID %s stopped error: %w", id, err)
	}
	return nil
}

func (s *Scheduler) TriggerTask(id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	task, exists := s.tasks[id]

	if !exists {
		return fmt.Errorf("task with ID %s not found", id)
	}

	if task.runner.running {
		return fmt.Errorf("task with ID %s is already running", id)
	}
	go func() {
		task.runner.start()
	}()
	return nil
}

func (s *Scheduler) EnableTaskJob(id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	task, exists := s.tasks[id]

	if !exists {
		return fmt.Errorf("task with ID %s not found", id)
	}

	if task.enabled {
		return fmt.Errorf("task with ID %s is already enabled", id)
	}

	task.enabled = true
	go func() {
		ticker := time.NewTicker(task.Interval)
		for {
			select {
			case <-ticker.C:
				task.runner.start()
			case <-task.stop:
				ticker.Stop()
				return
			}
		}
	}()

	return nil
}

func (s *Scheduler) DisableTask(id string) error {
	s.mutex.Lock()
	task, exists := s.tasks[id]
	s.mutex.Unlock()

	if !exists {
		return fmt.Errorf("task with ID %s not found", id)
	}

	if !task.enabled {
		return fmt.Errorf("task with ID %s is already disabled", id)
	}

	task.stop <- true
	task.enabled = false
	return nil
}

func (s *Scheduler) handleTriggerTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("taskID")

	err := s.TriggerTask(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	slog.Info("Task triggered", "id", id)
}

func (s *Scheduler) handleStopTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("taskID")

	err := s.StopTask(context.Background(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	slog.Info("Task stopped", "id", id)
}

func (s *Scheduler) handleEnableTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("taskID")

	err := s.EnableTaskJob(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	slog.Info("Task enabled", "id", id)
}

func (s *Scheduler) handleDisableTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("taskID")

	err := s.DisableTask(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	slog.Info("Task disabled", "id", id)
}

// RegisterRoutes registers the scheduler routes to the given mux
func (s *Scheduler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /scheduler/{taskID}/enable", s.handleEnableTask)
	mux.HandleFunc("POST /scheduler/{taskID}/disable", s.handleDisableTask)
	mux.HandleFunc("POST /scheduler/{taskID}/trigger", s.handleTriggerTask)
	mux.HandleFunc("POST /scheduler/{taskID}/stop", s.handleStopTask)
}
