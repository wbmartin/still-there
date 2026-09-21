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
		fmt.Fprintf(
			os.Stderr,
			"Failed to load configuration: %s\n",
			err.Error(),
		)
		os.Exit(1)
	}

	// Initialize key-value store (in-process "memory" by default; "shrmpl" for the external service)
	kv := app_lib.NewKVFromBackend(config.KVBackend, config.KVHostPort)
	defer kv.Close()

	// Create the logger (it will handle shrmpl-log internally)
	logger := app_lib.NewLogger(config.ServerName, config.LogReceiverHostPort)
	defer logger.Close()
	logger.Info("SERVER_INIT", "Starting API server", "username", "app-root")

	// Initialize database
	dbPool, err := pgxpool.New(context.Background(), config.DatabaseURL)
	if err != nil {
		logger.Error("DBI0",
			"Failed to create database connection pool: "+err.Error(),
			"username", "app-root")
		os.Exit(1)
	}
	defer dbPool.Close()
	logger.Info("DB_INIT", "Database connected", "username", "app-root")

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
	refreshHandler := handler.RefreshHandler(
		appServices,
		userRepo,
		appUserGroupRepo,
	)
	changeSecurityContextHandler := handler.ChangeSecurityContextHandler(
		appServices,
		userRepo,
		appUserGroupRepo,
	)
	changeOwnPasswordHandler := handler.ChangeOwnPasswordHandler(
		appServices,
		userRepo,
	)
	logoffHandler := handler.LogoffHandler(appServices)
	exampleHandler := handler.NewExampleHandler(appServices, exampleRepo)
	userHandler := handler.NewUserHandler(
		appServices,
		userRepo,
		appUserGroupRepo,
	)
	appUserGroupHandler := handler.NewAppUserGroupHandler(
		appServices,
		appUserGroupRepo,
	)
	errorReportHandler := handler.NewErrorReportHandler(appServices)
	userSelfCreateHandler := handler.UserSelfCreateHandler(
		appServices,
		userRepo,
	)
	mfaSetupHandler := handler.MFASetupHandler(appServices, userRepo)
	mfaVerifyHandler := handler.MFAVerifyHandler(appServices, userRepo)

	// Setup routes & map handlers
	mux := http.NewServeMux()

	// Apply security headers middleware to all routes
	securedMux := app_lib.SecurityHeadersMiddleware(mux)

	// Apply CORS middleware if origins are configured
	if len(config.CorsAllowedOrigins) > 0 {
		securedMux = app_lib.CORSMiddleware(
			config.CorsAllowedOrigins,
		)(
			securedMux,
		)
	}

	// Health check
	mux.HandleFunc("GET /health", healthHandler)

	// Authentication
	mux.HandleFunc("POST /auth", authHandler)
	mux.HandleFunc("GET /refresh", refreshHandler)
	mux.HandleFunc("POST /changeSecurityContext", changeSecurityContextHandler)
	mux.HandleFunc("POST /changeOwnPassword", changeOwnPasswordHandler)
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

	// User self-create (public)
	mux.HandleFunc("POST /userSelfCreate", userSelfCreateHandler)

	// MFA
	mux.HandleFunc("POST /mfaSetup", mfaSetupHandler)
	mux.HandleFunc("POST /mfaVerify", mfaVerifyHandler)

	// User CRUD
	mux.HandleFunc("GET /userList", userHandler.RetrieveUserList)
	mux.HandleFunc("POST /user", userHandler.CreateUser)
	mux.HandleFunc("PUT /user", userHandler.UpdateUser)
	mux.HandleFunc("DELETE /user", userHandler.DeleteUser)
	mux.HandleFunc(
		"POST /changeOtherUserPassword",
		userHandler.ChangeOtherUserPassword,
	)
	mux.HandleFunc("POST /addUserListToGroup",
		userHandler.AddUserListToGroupHandler)
	mux.HandleFunc("POST /removeUserListFromGroup",
		userHandler.RemoveUserListFromGroupHandler)

	// Admin reset MFA
	adminResetMFAHandler := handler.AdminResetUserMFAHandler(
		appServices,
		userRepo,
	)
	mux.HandleFunc("POST /admin/resetUserMFA", adminResetMFAHandler)

	// Receive Grant list is available, but shouldn't be used in most cases.  Clients will have their own copy of it.
	// This is an intentional coupling point, mitigated by most changes being additive only.
	mux.HandleFunc("GET /grantList", userHandler.RetrieveGrantList)

	// User groups
	mux.HandleFunc(
		"GET /appUserGroupList",
		appUserGroupHandler.RetrieveAppUserGroupList,
	)
	mux.HandleFunc("POST /appUserGroup", appUserGroupHandler.CreateAppUserGroup)
	mux.HandleFunc("PUT /appUserGroup", appUserGroupHandler.UpdateAppUserGroup)
	mux.HandleFunc(
		"DELETE /appUserGroup",
		appUserGroupHandler.DeleteAppUserGroup,
	)

	// Error reporting (no auth required)
	mux.HandleFunc("POST /error-report", errorReportHandler.HandleErrorReport)

	// Create server
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.ServerBindAddress, config.ServerPort),
		Handler: securedMux,
	}

	// Start server in goroutine
	go func() {
		if config.EnableTLS {
			logger.Info(
				"SERVER_START",
				fmt.Sprintf(
					"Server starting on HTTPS port: %d",
					config.ServerPort,
				),
				"username",
				"app-root",
			)
			err := server.ListenAndServeTLS(
				config.TLSCertPath,
				config.TLSKeyPath,
			)
			if err != nil && err != http.ErrServerClosed {
				logger.Error("STF0",
					"Server failed to start: "+err.Error(),
					"username", "app-root")
				os.Exit(1)
			}
		} else {
			logger.Info("SERVER_START",
				fmt.Sprintf("Server starting on HTTP port: %d", config.ServerPort),
				"username", "app-root")
			err := server.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				logger.Error("STF0",
					"Server failed to start: "+err.Error(),
					"username", "app-root")
				os.Exit(1)
			}
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(
		"SERVER_SHUTDOWN",
		"Shutting down server",
		"username",
		"app-root",
	)

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("SHF0",
			"Server forced to shutdown: "+err.Error(),
			"username", "app-root")
		os.Exit(1)
	}

	logger.Info("SERVER_STOP", "Server stopped", "username", "app-root")
}
