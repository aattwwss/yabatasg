package scheduler

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type Task struct {
	ID       string
	Interval time.Duration
	Execute  func()
	stop     chan bool
	running  bool
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

func (s *Scheduler) AddTask(id string, interval time.Duration, t func()) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	task := &Task{
		ID:       id,
		Interval: interval,
		Execute:  t,
		stop:     make(chan bool),
		running:  false,
	}
	s.tasks[id] = task
}

func (s *Scheduler) TriggerTask(id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	task, exists := s.tasks[id]

	if !exists {
		return fmt.Errorf("task with ID %s not found", id)
	}

	if task.running {
		return fmt.Errorf("task with ID %s is already running", id)
	}
	go func() {
		task.mutex.Lock()
		task.running = true
		task.mutex.Unlock()

		task.Execute()

		task.mutex.Lock()
		task.running = false
		task.mutex.Unlock()
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

	if task.running {
		return fmt.Errorf("task with ID %s is already running", id)
	}

	task.enabled = true
	go func() {
		ticker := time.NewTicker(task.Interval)
		for {
			select {
			case <-ticker.C:
				task.mutex.Lock()
				task.running = true
				task.mutex.Unlock()

				task.Execute()

				task.mutex.Lock()
				task.running = false
				task.mutex.Unlock()
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
}
