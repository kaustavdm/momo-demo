package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kaustavdm/twilio-momo/internal/types"
	"github.com/nats-io/nats.go"
)

type Scheduler struct {
	nc     *nats.Conn
	tasks  map[string]types.Task
	mutex  sync.RWMutex
	logger *log.Logger
}

func NewScheduler(nc *nats.Conn) *Scheduler {
	return &Scheduler{
		nc:     nc,
		tasks:  make(map[string]types.Task),
		logger: log.New(log.Writer(), "[SCHEDULER] ", log.LstdFlags),
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	// Subscribe to task status updates
	_, err := s.nc.Subscribe("task.status", func(msg *nats.Msg) {
		var result types.TaskResult
		if err := json.Unmarshal(msg.Data, &result); err != nil {
			s.logger.Printf("Error unmarshaling task result: %v", err)
			return
		}

		s.mutex.Lock()
		if task, exists := s.tasks[result.TaskID]; exists {
			task.Status = result.Status
			task.LastRun = result.Timestamp
			s.tasks[result.TaskID] = task
		}
		s.mutex.Unlock()
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe to task status: %v", err)
	}

	// Start scheduler loop
	go s.scheduleLoop(ctx)

	return nil
}

func (s *Scheduler) scheduleLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkAndDispatchTasks()
		}
	}
}

func (s *Scheduler) checkAndDispatchTasks() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	for id, task := range s.tasks {
		if task.Status == types.TaskStatusPending && !task.ScheduledAt.After(now) {
			task.Status = types.TaskStatusRunning
			s.tasks[id] = task

			taskData, _ := json.Marshal(task)
			if err := s.nc.Publish("task.dispatch", taskData); err != nil {
				s.logger.Printf("Error dispatching task %s: %v", id, err)
				task.Status = types.TaskStatusFailed
				s.tasks[id] = task
			}
		}
	}
}

func (s *Scheduler) AddTask(task types.Task) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if task.ID == "" {
		task.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.Status == "" {
		task.Status = types.TaskStatusPending
	}

	s.tasks[task.ID] = task
	return nil
}

func (s *Scheduler) GetTasks() []types.Task {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	tasks := make([]types.Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

func (s *Scheduler) Stop() error {
	return nil
}
