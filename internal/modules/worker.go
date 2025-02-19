package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/kaustavdm/twilio-momo/internal/types"
	"github.com/nats-io/nats.go"
)

type Worker struct {
	nc     *nats.Conn
	logger *log.Logger
}

func NewWorker(nc *nats.Conn) *Worker {
	return &Worker{
		nc:     nc,
		logger: log.New(log.Writer(), "[WORKER] ", log.LstdFlags),
	}
}

func (w *Worker) Start(ctx context.Context) error {
	// Subscribe to task dispatch
	_, err := w.nc.Subscribe("task.dispatch", func(msg *nats.Msg) {
		var task types.Task
		if err := json.Unmarshal(msg.Data, &task); err != nil {
			w.logger.Printf("Error unmarshaling task: %v", err)
			return
		}

		startTime := time.Now()
		w.logger.Printf("Executing task: %s", task.Name)

		// Simulate task execution
		time.Sleep(2 * time.Second)

		endTime := time.Now()
		duration := endTime.Sub(startTime).Seconds()

		result := types.TaskResult{
			TaskID:    task.ID,
			Status:    types.TaskStatusCompleted,
			Output:    fmt.Sprintf("Executed task: %s", task.Name),
			Timestamp: endTime,
			Duration:  duration,
		}

		resultData, _ := json.Marshal(result)
		if err := w.nc.Publish("task.status", resultData); err != nil {
			w.logger.Printf("Error publishing task result: %v", err)
		}
	})

	return err
}

func (w *Worker) Stop() error {
	return nil
}
