package types

import "time"

type Task struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Payload     string    `json:"payload"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Status      string    `json:"status"`
	Priority    int       `json:"priority"`
	Retries     int       `json:"retries"`
	LastRun     time.Time `json:"last_run,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type TaskResult struct {
	TaskID    string    `json:"task_id"`
	Status    string    `json:"status"`
	Output    string    `json:"output"`
	Timestamp time.Time `json:"timestamp"`
	Duration  float64   `json:"duration"`
	Error     string    `json:"error,omitempty"`
}

// Possible task statuses
const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusFailed    = "failed"
)
