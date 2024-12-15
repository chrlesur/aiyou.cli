// Package main is the entry point for the AI.YOU CLI application.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/chrlesur/aiyou.cli/internal/cli"
	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/sirupsen/logrus"
)

// main initializes and starts the AI.YOU CLI application.
// It handles graceful shutdown and proper error management.
func main() {
	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	if cfg.Debug {
		logger.SetLevel(logrus.DebugLevel)
	}
	logger.Info("Starting AI.YOU CLI")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize CLI application
	app, err := cli.NewApp(&cli.AppConfig{
		Version: "1.0.0", // Version can be set from build flags
		Config:  cfg,
		Logger:  logger,
	})
	if err != nil {
		logger.Fatalf("Failed to initialize CLI application: %v", err)
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
			logger.Errorf("Application error: %v", err)
			os.Exit(1)
		}
	case <-sigChan:
		logger.Info("Shutting down gracefully...")
		cancel()
		// Wait for the application to finish
		<-done
	}
}
