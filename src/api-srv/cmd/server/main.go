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
	defer kv.Close()

	// Create the logger (it will handle shrmpl-log internally)
	logger := app_lib.NewLogger(config)
	defer logger.Close()
	logger.Info("INI0", "Starting API server", "username", "app-root")

	// Initialize database
	dbPool, err := pgxpool.New(context.Background(), config.DatabaseURL)
	if err != nil {
		logger.Error("DBI0",
			"Failed to create database connection pool: "+err.Error(),
			"username", "app-root")
		os.Exit(1)
	}
	defer dbPool.Close()
	logger.Info("DBI1", "Database connected", "username", "app-root")

	// Test the connection
	if err := dbPool.Ping(context.Background()); err != nil {
		logger.Error("DBP0",
			"Failed to ping database: "+err.Error(),
			"username", "app-root")
		os.Exit(1)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(dbPool)
	appUserGroupRepo := repository.NewAppUserGroupRepository(dbPool)
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
	authHandler := handler.AuthHandler(appServices, userRepo, appUserGroupRepo)
	refreshHandler := handler.RefreshHandler(appServices, userRepo, appUserGroupRepo)
	changeSecurityContextHandler := handler.ChangeSecurityContextHandler(appServices, userRepo, appUserGroupRepo)
	logoffHandler := handler.LogoffHandler(appServices)

	exampleHandler := handler.NewExampleHandler(appServices, exampleRepo)
	userHandler := handler.NewUserHandler(appServices, userRepo)
	appUserGroupHandler := handler.NewAppUserGroupHandler(appServices, appUserGroupRepo)
	errorReportHandler := handler.NewErrorReportHandler(appServices)

	// Setup routes
	mux := http.NewServeMux()

	// Apply security headers middleware to all routes
	securedMux := app_lib.SecurityHeadersMiddleware(mux)

	// Health check
	mux.HandleFunc("GET /health", healthHandler)

	// Authentication
	mux.HandleFunc("POST /auth", authHandler)
	mux.HandleFunc("GET /refresh", refreshHandler)
	mux.HandleFunc("POST /changeSecurityContext", changeSecurityContextHandler)
	mux.HandleFunc("POST /logoff", logoffHandler)

	// Example CRUD
	mux.HandleFunc("GET /example", exampleHandler.RetrieveExample)
	mux.HandleFunc("POST /example", exampleHandler.CreateExample)
	mux.HandleFunc("PUT /example", exampleHandler.UpdateExample)
	mux.HandleFunc("DELETE /example", exampleHandler.DeleteExample)

	// Example list (all or by IDs)
	mux.HandleFunc("GET /exampleList", exampleHandler.RetrieveExampleList)
	mux.HandleFunc("POST /exampleList", exampleHandler.CreateExampleList)
	mux.HandleFunc("PUT /exampleList", exampleHandler.UpdateExampleList)
	mux.HandleFunc("DELETE /exampleList", exampleHandler.DeleteExampleList)

	// User CRUD
	mux.HandleFunc("GET /userList", userHandler.RetrieveUserList)
	mux.HandleFunc("POST /user", userHandler.CreateUser)
	mux.HandleFunc("PUT /user", userHandler.UpdateUser)
	mux.HandleFunc("DELETE /user", userHandler.DeleteUser)
	mux.HandleFunc("POST /addUserListToGroup",
		userHandler.AddUserListToGroupHandler)
	mux.HandleFunc("POST /removeUserListFromGroup",
		userHandler.RemoveUserListFromGroupHandler)

	mux.HandleFunc("GET /grantList", userHandler.RetrieveGrantList)

	// User groups
	mux.HandleFunc("GET /appUserGroupList", appUserGroupHandler.RetrieveAppUserGroupList)
	mux.HandleFunc("POST /appUserGroup", appUserGroupHandler.CreateAppUserGroup)
	mux.HandleFunc("PUT /appUserGroup", appUserGroupHandler.UpdateAppUserGroup)
	mux.HandleFunc("DELETE /appUserGroup", appUserGroupHandler.DeleteAppUserGroup)

	// Error reporting (no auth required)
	mux.HandleFunc("POST /error-report", errorReportHandler.HandleErrorReport)

	// Create server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.ServerPort),
		Handler: securedMux,
	}

	// Start server in goroutine
	go func() {
		logger.Info("STR0",
			fmt.Sprintf("Server starting on HTTPS port: %d", config.ServerPort),
			"username", "app-root")
		err := server.ListenAndServeTLS(config.TLSCertPath, config.TLSKeyPath)
		if err != nil && err != http.ErrServerClosed {
			logger.Error("STF0",
				"Server failed to start: "+err.Error(),
				"username", "app-root")
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("SHD0", "Shutting down server", "username", "app-root")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("SHF0",
			"Server forced to shutdown: "+err.Error(),
			"username", "app-root")
		os.Exit(1)
	}

	logger.Info("STP0", "Server stopped", "username", "app-root")
}
