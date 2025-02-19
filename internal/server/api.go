package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/kaustavdm/twilio-momo/internal/modules"
	"github.com/kaustavdm/twilio-momo/internal/types"
)

type APIServer struct {
	port      string
	scheduler *modules.Scheduler
	reporter  *modules.Reporter
	server    *http.Server
	logger    *log.Logger
}

// APIError represents an error response
type APIError struct {
	Error string `json:"error"`
}

// APIResponse represents a success response
type APIResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func NewAPIServer(port string, scheduler *modules.Scheduler) *APIServer {
	return &APIServer{
		port:      port,
		scheduler: scheduler,
		logger:    log.New(log.Writer(), "[API] ", log.LstdFlags),
	}
}

func (s *APIServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/health", s.handleHealth())
	mux.HandleFunc("/tasks", s.handleTasks())
	mux.HandleFunc("/tasks/", s.handleTaskByID()) // For future use with task ID
	mux.HandleFunc("/metrics", s.handleMetrics())

	s.server = &http.Server{
		Addr:    ":" + s.port,
		Handler: s.logRequest(mux),
	}

	// Start server
	go func() {
		s.logger.Printf("API server starting on port %s", s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Printf("API server error: %v", err)
		}
	}()

	// Wait for context cancellation to stop server
	<-ctx.Done()
	return s.Stop()
}

func (s *APIServer) Stop() error {
	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.logger.Println("Shutting down API server...")
	return s.server.Shutdown(ctx)
}

// Middleware for logging requests
func (s *APIServer) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		s.logger.Printf("Started %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		s.logger.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// Health check handler
func (s *APIServer) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		s.writeJSON(w, APIResponse{
			Message: "OK",
			Data: map[string]string{
				"status": "healthy",
				"time":   time.Now().Format(time.RFC3339),
			},
		})
	}
}

// Tasks handler
func (s *APIServer) handleTasks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// List tasks
			tasks := s.scheduler.GetTasks()
			s.writeJSON(w, APIResponse{
				Message: "Tasks retrieved successfully",
				Data:    tasks,
			})

		case http.MethodPost:
			// Create new task
			var task types.Task
			if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
				s.writeError(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			// Validate task
			if err := s.validateTask(task); err != nil {
				s.writeError(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Set defaults
			task.CreatedAt = time.Now()
			if task.Status == "" {
				task.Status = types.TaskStatusPending
			}

			// Add task to scheduler
			if err := s.scheduler.AddTask(task); err != nil {
				s.writeError(w, "Failed to create task", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
			s.writeJSON(w, APIResponse{
				Message: "Task created successfully",
				Data:    task,
			})

		default:
			s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// Task by ID handler (for future use)
func (s *APIServer) handleTaskByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement task by ID operations
		s.writeError(w, "Not implemented", http.StatusNotImplemented)
	}
}

// Metrics handler
func (s *APIServer) handleMetrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if s.reporter == nil {
			s.writeError(w, "Metrics not available", http.StatusServiceUnavailable)
			return
		}

		metrics := s.reporter.GetMetrics()
		s.writeJSON(w, APIResponse{
			Message: "Metrics retrieved successfully",
			Data:    metrics,
		})
	}
}

// Helper function to validate task
func (s *APIServer) validateTask(task types.Task) error {
	if task.Name == "" {
		return fmt.Errorf("task name is required")
	}
	if task.ScheduledAt.IsZero() {
		return fmt.Errorf("scheduled time is required")
	}
	if task.ScheduledAt.Before(time.Now()) {
		return fmt.Errorf("scheduled time must be in the future")
	}
	return nil
}

// Helper function to write JSON response
func (s *APIServer) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Printf("Error encoding JSON: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// Helper function to write error response
func (s *APIServer) writeError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIError{Error: message})
}
