package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/kaustavdm/twilio-momo/internal/modules"
	"github.com/kaustavdm/twilio-momo/internal/server"
	"github.com/nats-io/nats.go"
)

var (
	Version   string
	BuildTime string
	GitCommit string
)

func main() {
	// Command line flags
	moduleList := flag.String("modules", "all", "Comma-separated list of modules to run (scheduler,worker,reporter,archiver,api) or 'all'")
	natsURL := flag.String("nats-url", nats.DefaultURL, "NATS server URL")
	httpPort := flag.String("port", "8080", "HTTP port for API")
	flag.Parse()

	log.Printf("Starting momo %s (Built: %s, Commit: %s)", Version, BuildTime, GitCommit)

	// Connect to NATS
	nc, err := nats.Connect(*natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize module registry
	moduleRegistry := make(map[string]modules.Module)
	moduleRegistry["scheduler"] = modules.NewScheduler(nc)
	moduleRegistry["worker"] = modules.NewWorker(nc)
	moduleRegistry["reporter"] = modules.NewReporter(nc)
	// moduleRegistry["archiver"] = modules.NewArchiver(nc)

	// Determine which modules to run
	var modulesToRun []string
	if *moduleList == "all" {
		modulesToRun = []string{"scheduler", "worker", "reporter", "archiver", "api"}
	} else {
		modulesToRun = strings.Split(*moduleList, ",")
	}

	// Start selected modules
	var wg sync.WaitGroup
	for _, name := range modulesToRun {
		if name == "api" {
			// Start API server if requested
			apiServer := server.NewAPIServer(*httpPort, moduleRegistry["scheduler"].(*modules.Scheduler))
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := apiServer.Start(ctx); err != nil {
					log.Printf("API server error: %v", err)
				}
			}()
			continue
		}

		if module, exists := moduleRegistry[name]; exists {
			wg.Add(1)
			go func(m modules.Module, name string) {
				defer wg.Done()
				log.Printf("Starting module: %s", name)
				if err := m.Start(ctx); err != nil {
					log.Printf("Module %s error: %v", name, err)
				}
			}(module, name)
		}
	}

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	cancel()
	wg.Wait()
}
