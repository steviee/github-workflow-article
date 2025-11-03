// Package main is the entry point for the image processing REST API server.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"

	"github.com/steviee/github-workflow-article/internal/config"
	"github.com/steviee/github-workflow-article/internal/handler"
	appMiddleware "github.com/steviee/github-workflow-article/internal/middleware"
)

func main() {
	// Initialize logger with JSON formatting for production
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	logger.SetOutput(os.Stdout)

	// Load configuration from environment
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Set log level from configuration
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logger.WithError(err).Warn("Invalid log level, defaulting to info")
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Log startup message with configuration
	logger.WithFields(logrus.Fields{
		"port":                   cfg.Port,
		"cache_ttl":              cfg.CacheTTL,
		"max_image_size":         cfg.MaxImageSize,
		"max_output_dimension":   cfg.MaxOutputDimension,
		"log_level":              cfg.LogLevel,
		"cache_cleanup_interval": cfg.CacheCleanupInterval,
	}).Info("Image Processing API starting...")

	// Create chi router
	router := chi.NewRouter()

	// Add middleware in order
	router.Use(appMiddleware.RequestLogger(logger))
	router.Use(appMiddleware.CORS)
	router.Use(appMiddleware.Metrics)
	router.Use(middleware.Recoverer)                 // Recover from panics
	router.Use(middleware.Timeout(30 * time.Second)) // Request timeout

	// Register routes
	router.Get("/health", handler.HealthHandler)
	router.Get("/ready", handler.ReadyHandler)
	router.Handle("/metrics", handler.MetricsHandler())
	router.Get("/image", handler.ImageHandler)

	// Create HTTP server with timeouts
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("addr", server.Addr).Info("HTTP server listening")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("HTTP server failed")
		}
	}()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Block until signal received
	sig := <-sigChan
	logger.WithField("signal", sig.String()).Info("Shutdown signal received")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Attempt graceful shutdown
	logger.Info("Initiating graceful shutdown...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Error during shutdown")
		os.Exit(1)
	}

	logger.Info("Server shutdown complete")
	fmt.Println("Server shutdown complete")
}
