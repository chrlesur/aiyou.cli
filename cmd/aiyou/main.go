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
	logger := cli.NewLogger(cfg.LogLevel)
	logger.Info("Starting AI.YOU CLI")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize CLI application
	app, err := cli.NewApp(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to initialize CLI application: %v", err)
	}

	// Start the application
	go func() {
		if err := app.Run(ctx); err != nil {
			logger.Errorf("Application error: %v", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutting down gracefully...")
	cancel()
}
