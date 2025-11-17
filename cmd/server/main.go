package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang-backend-service/internal/api"
	"golang-backend-service/internal/config"
	"golang-backend-service/internal/database"
	"golang-backend-service/internal/logger"
	"golang-backend-service/internal/reputation"

	_ "golang-backend-service/docs"

	"github.com/sirupsen/logrus"
)

// @title GoLang Backend Service API
// @version 1.0
// @description A template backend service built with GoLang featuring REST API, PostgreSQL, logging, and monitoring
// @contact.name API Support
// @contact.email support@example.com
// @host localhost:8080
// @BasePath /
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(cfg.Logger.Level); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting GoLang Backend Service")
	logger.WithFields(logrus.Fields{
		"version": "1.0.0",
		"port":    cfg.Server.Port,
	}).Info("Service configuration loaded")

	// Connect to database
	dsn := cfg.GetDatabaseDSN()
	if err := database.Connect(dsn, logger.Log); err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("Failed to connect to database")
	}
	defer database.Close()

	// Set up routes
	router := api.SetupRoutes()

	// Start IP reputation aggregation service
	reputationConfig := reputation.DefaultReputationConfig()
	aggregationService := reputation.NewAggregationService(reputationConfig)
	if err := aggregationService.Start(5); err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Warn("Failed to start IP reputation aggregation service")
	} else {
		logger.Info("IP reputation aggregation service started")
	}
	defer aggregationService.Stop()

	// Create HTTP server
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.WithFields(logrus.Fields{
			"address": addr,
		}).Info("HTTP server starting")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Fatal("Failed to start server")
		}
	}()

	logger.WithFields(logrus.Fields{
		"address": addr,
		"swagger": fmt.Sprintf("http://localhost%s/swagger/index.html", addr),
		"metrics": fmt.Sprintf("http://localhost%s/metrics", addr),
	}).Info("Service is ready")

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Server forced to shutdown")
	}

	logger.Info("Server exited gracefully")
}

