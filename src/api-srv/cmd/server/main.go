package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/handler"
	"github.com/wbmartin/starter-api/internal/repository"
)

func main() {
	// Parse command line arguments first to get config file
	configFile := flag.String("config", "etc/loc-dev-config.json", "Path to configuration file")
	flag.Parse()

	// Load configuration
	config, err := app_lib.LoadConfig(*configFile)
	if err != nil {
		// Use fallback logger for early errors
		fallbackLogger := app_lib.NewFallbackLogger("server")
		fallbackLogger.Error("Failed to load configuration: %s", err.Error())
		os.Exit(1)
	}

	// Initialize shrmpl clients first
	shrmplKV := app_lib.NewShrmplKVClient(config.ShrmplKVHost, config.ShrmplKVPort)
	if err := shrmplKV.Connect(); err != nil {
		fallbackLogger := app_lib.NewFallbackLogger("server")
		fallbackLogger.Error("Failed to connect to shrmpl-kv: %s", err.Error())
		os.Exit(1)
	}
	defer shrmplKV.Close()

	shrmplLog, err := app_lib.NewShrmplLogClient(config.ShrmplLogDest)
	if err != nil {
		fallbackLogger := app_lib.NewFallbackLogger("server")
		fallbackLogger.Error("Failed to create shrmpl-log client: %s", err.Error())
		os.Exit(1)
	}
	if err := shrmplLog.Connect(); err != nil {
		fallbackLogger := app_lib.NewFallbackLogger("server")
		fallbackLogger.Error("Failed to connect to shrmpl-log: %s", err.Error())
		os.Exit(1)
	}
	defer shrmplLog.Close()

	// Now create the proper logger
	logger := app_lib.NewLogger("server", shrmplLog)
	logger.Info("Starting API server")

	// Initialize database
	db, err := app_lib.NewDatabase(config.DatabaseURL)
	if err != nil {
		logger.Error("Failed to initialize database: %s", err.Error())
		os.Exit(1)
	}
	defer db.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db.Pool)
	userGroupRepo := repository.NewUserGroupRepository(db.Pool)
	exampleRepo := repository.NewExampleRepository(db.Pool)

	// Initialize handlers
	healthHandler := handler.HealthHandler(logger.With("component", "health"))
	authHandler := handler.AuthHandler(db, shrmplKV, config, logger.With("component", "auth"), userRepo, userGroupRepo)
	auth2Handler := handler.Auth2Handler(db, shrmplKV, config, logger.With("component", "auth2"), userRepo, userGroupRepo)
	logoffHandler := handler.LogoffHandler(logger.With("component", "logoff"))

	exampleHandler := handler.NewExampleHandler(db, shrmplKV, config, logger.With("component", "example"), exampleRepo)

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

	// Create server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.ServerPort),
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Server starting on port: %d", config.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start: %s", err.Error())
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %s", err.Error())
		os.Exit(1)
	}

	logger.Info("Server stopped")
}
