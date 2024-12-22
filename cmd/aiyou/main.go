// Package main is the entry point for the AI.YOU CLI application.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/chrlesur/aiyou.cli/internal/cli"
	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/chrlesur/aiyou.cli/pkg/logger"
	"github.com/joho/godotenv"
)

// main initializes and starts the AI.YOU CLI application.
// It handles graceful shutdown and proper error management.
func main() {
	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Check command line arguments for verbose/debug flags
	isVerbose := false
	isDebug := false
	for _, arg := range os.Args {
		switch arg {
		case "--verbose":
			isVerbose = true
		case "--debug":
			isDebug = true
		}
	}

	// Initialize logger with appropriate mode
	log := logger.GetLogger()
	var logLevel logger.LogLevel
	var silent bool

	switch {
	case isDebug:
		logLevel = logger.DebugLevel
		silent = false
	case isVerbose:
		logLevel = logger.InfoLevel
		silent = false
	default:
		logLevel = logger.InfoLevel
		silent = true
	}

	err := log.Configure(logger.Config{
		LogDir: "logs",
		Level:  logLevel,
		Silent: silent,
	})
	if err != nil {
		// Fallback to direct stdout since logger isn't configured yet
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Load .env file if present
	if err := godotenv.Load(); err != nil {
		log.Debug("No .env file found or error loading it: %v", err)
	} else {
		log.Debug("Successfully loaded .env file")
	}

	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		log.Error("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	// Update config based on command line flags
	if isDebug {
		cfg.Debug = true
		log.Debug("Debug mode enabled")
	}
	if isVerbose {
		log.Info("Verbose mode enabled")
	}

	log.Info("Starting AI.YOU CLI")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize CLI application
	app, err := cli.NewApp(&cli.AppConfig{
		Version: "1.0.0", // Version can be set from build flags
		Config:  cfg,
	})
	if err != nil {
		log.Error("Failed to initialize CLI application: %v", err)
		os.Exit(1)
	}

	// Create a channel to receive run completion
	done := make(chan error, 1)

	// Start the application
	go func() {
		done <- app.Run(ctx)
	}()

	// Wait for either command completion or shutdown signal
	select {
	case err := <-done:
		if err != nil {
			log.Error("Application error: %v", err)
			os.Exit(1)
		}
	case <-sigChan:
		log.Info("Shutting down gracefully...")
		cancel()
		// Wait for the application to finish
		<-done
	}

	log.Info("Application shutdown complete")
}
