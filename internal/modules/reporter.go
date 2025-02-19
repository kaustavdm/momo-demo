// internal/modules/reporter.go
package modules

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/kaustavdm/twilio-momo/internal/types"
	"github.com/nats-io/nats.go"
)

type Reporter struct {
	nc           *nats.Conn
	logger       *log.Logger
	metrics      map[string]TaskMetrics
	metricsMutex sync.RWMutex
}

type TaskMetrics struct {
	TotalTasks      int       `json:"total_tasks"`
	CompletedTasks  int       `json:"completed_tasks"`
	FailedTasks     int       `json:"failed_tasks"`
	AverageDuration float64   `json:"average_duration"`
	LastUpdated     time.Time `json:"last_updated"`
	TotalDuration   float64   `json:"-"` // Used for average calculation
}

func NewReporter(nc *nats.Conn) *Reporter {
	return &Reporter{
		nc:      nc,
		logger:  log.New(log.Writer(), "[REPORTER] ", log.LstdFlags),
		metrics: make(map[string]TaskMetrics),
	}
}

func (r *Reporter) Start(ctx context.Context) error {
	// Subscribe to task status updates for metrics
	_, err := r.nc.Subscribe("task.status", func(msg *nats.Msg) {
		var result types.TaskResult
		if err := json.Unmarshal(msg.Data, &result); err != nil {
			r.logger.Printf("Error unmarshaling task result: %v", err)
			return
		}

		r.updateMetrics(result)
	})

	if err != nil {
		return err
	}

	// Start periodic metrics publishing
	go r.publishMetrics(ctx)

	// Start periodic metrics cleanup
	go r.cleanupMetrics(ctx)

	return nil
}

func (r *Reporter) updateMetrics(result types.TaskResult) {
	r.metricsMutex.Lock()
	defer r.metricsMutex.Unlock()

	// Get current date as key for daily metrics
	dateKey := result.Timestamp.Format("2006-01-02")

	metrics, exists := r.metrics[dateKey]
	if !exists {
		metrics = TaskMetrics{
			LastUpdated: result.Timestamp,
		}
	}

	// Update metrics based on task result
	metrics.TotalTasks++
	metrics.LastUpdated = result.Timestamp
	metrics.TotalDuration += result.Duration
	metrics.AverageDuration = metrics.TotalDuration / float64(metrics.TotalTasks)

	switch result.Status {
	case types.TaskStatusCompleted:
		metrics.CompletedTasks++
	case types.TaskStatusFailed:
		metrics.FailedTasks++
	}

	r.metrics[dateKey] = metrics
}

func (r *Reporter) publishMetrics(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.metricsMutex.RLock()

			// Create metrics report
			report := make(map[string]TaskMetrics)
			for date, metrics := range r.metrics {
				report[date] = metrics
			}

			r.metricsMutex.RUnlock()

			// Publish metrics
			if reportData, err := json.Marshal(report); err == nil {
				if err := r.nc.Publish("metrics.report", reportData); err != nil {
					r.logger.Printf("Error publishing metrics: %v", err)
				}
			} else {
				r.logger.Printf("Error marshaling metrics: %v", err)
			}
		}
	}
}

func (r *Reporter) cleanupMetrics(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.metricsMutex.Lock()

			// Keep only last 30 days of metrics
			cutoffDate := time.Now().AddDate(0, 0, -30)
			for date := range r.metrics {
				if parsedDate, err := time.Parse("2006-01-02", date); err == nil {
					if parsedDate.Before(cutoffDate) {
						delete(r.metrics, date)
					}
				}
			}

			r.metricsMutex.Unlock()
		}
	}
}

func (r *Reporter) GetMetrics() map[string]TaskMetrics {
	r.metricsMutex.RLock()
	defer r.metricsMutex.RUnlock()

	// Create a copy of metrics to return
	metrics := make(map[string]TaskMetrics)
	for date, metric := range r.metrics {
		metrics[date] = metric
	}

	return metrics
}

func (r *Reporter) Stop() error {
	// Publish final metrics before stopping
	r.metricsMutex.RLock()
	reportData, err := json.Marshal(r.metrics)
	r.metricsMutex.RUnlock()

	if err == nil {
		if err := r.nc.Publish("metrics.report", reportData); err != nil {
			r.logger.Printf("Error publishing final metrics: %v", err)
		}
	}

	return nil
}
