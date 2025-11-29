package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/handler"
	"github.com/wbmartin/starter-api/internal/repository"
)

func main() {
	// Parse command line arguments first to get config file
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Configuration file path is required\n")
		os.Exit(1)
	}

	configFile := os.Args[1]

	// Load configuration
	config, err := app_lib.LoadConfig(configFile)
	if err != nil {
		// Use console logging for early errors
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %s\n", err.Error())
		os.Exit(1)
	}

	// Initialize key-value store
	kv := app_lib.NewKV(config)

	// Create the logger (it will handle shrmpl-log internally)
	logger := app_lib.NewLogger(config)
	logger.Info("server", "INI0", "Starting API server")

	// Initialize database
	dbPool, err := pgxpool.New(context.Background(), config.DatabaseURL)
	if err != nil {
		logger.Error("server", "DBI0", "Failed to create database connection pool: "+err.Error())
		os.Exit(1)
	}
	defer dbPool.Close()
	logger.Error("server", "DBI1", "Database connected")

	// Test the connection
	if err := dbPool.Ping(context.Background()); err != nil {
		logger.Error("server", "DBP0", "Failed to ping database: "+err.Error())
		os.Exit(1)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(dbPool)
	userGroupRepo := repository.NewUserGroupRepository(dbPool)
	exampleRepo := repository.NewExampleRepository(dbPool)

	// Initialize app dependencies
	appServices := &app_lib.AppServices{
		DBPool: dbPool,
		KV:     kv,
		Config: config,
		Logger: logger,
	}

	// Initialize handlers
	healthHandler := handler.HealthHandler(appServices)
	authHandler := handler.AuthHandler(appServices, userRepo, userGroupRepo)
	auth2Handler := handler.Auth2Handler(appServices, userRepo, userGroupRepo)
	logoffHandler := handler.LogoffHandler(appServices)

	exampleHandler := handler.NewExampleHandler(appServices, exampleRepo)

	// Setup routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", healthHandler)

	// Authentication
	mux.HandleFunc("POST /auth", authHandler)
	mux.HandleFunc("POST /auth2", auth2Handler)
	mux.HandleFunc("POST /logoff", logoffHandler)

	// Example CRUD
	mux.HandleFunc("GET /example", exampleHandler.GetExample)
	mux.HandleFunc("POST /example", exampleHandler.CreateExample)
	mux.HandleFunc("PUT /example", exampleHandler.UpdateExample)
	mux.HandleFunc("DELETE /example", exampleHandler.DeleteExample)

	// Example batch operations
	mux.HandleFunc("GET /exampleList", exampleHandler.GetExampleList)
	mux.HandleFunc("POST /exampleList", exampleHandler.CreateExampleList)
	mux.HandleFunc("PUT /exampleList", exampleHandler.UpdateExampleList)
	mux.HandleFunc("DELETE /exampleList", exampleHandler.DeleteExampleList)

	// Create server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.ServerPort),
		Handler: mux,
	}

	// Start server in goroutine
	go func() {

		logger.Info("server", "STR0", fmt.Sprintf("Server starting on HTTPS port: %d", config.ServerPort))
		if err := server.ListenAndServeTLS(config.TLSCertPath, config.TLSKeyPath); err != nil && err != http.ErrServerClosed {
			logger.Error("server", "STF0", "Server failed to start: "+err.Error())
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("server", "SHD0", "Shutting down server")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server", "SHF0", "Server forced to shutdown: "+err.Error())
		os.Exit(1)
	}

	logger.Info("server", "STP0", "Server stopped")
}
