package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/cache"
	"github.com/wbmartin/starter-api/internal/config"
	"github.com/wbmartin/starter-api/internal/handler"
	"github.com/wbmartin/starter-api/internal/issuerelay"
	"github.com/wbmartin/starter-api/internal/logging"
	"github.com/wbmartin/starter-api/internal/repository"
	"github.com/wbmartin/starter-api/internal/security"
)

func main() {
	// Parse command line arguments first to get config file
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Configuration file path is required\n")
		os.Exit(1)
	}

	configFile := os.Args[1]

	// Load configuration
	config, err := config.LoadConfig(configFile)
	if err != nil {
		// Use console logging for early errors
		fmt.Fprintf(
			os.Stderr,
			"Failed to load configuration: %s\n",
			err.Error(),
		)
		os.Exit(1)
	}

	// In-process key-value store (rate limits, lockouts, MFA temp tokens,
	// session revocations)
	kv := cache.NewKV()
	defer kv.Close()

	// slog to stderr; systemd captures it into journald
	logger := logging.NewLogger(
		config.ServerName, os.Stderr, config.LogLevel,
	)
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
	helpEventRepository := repository.NewHelpEventRepository(dbPool)
	userInviteRepository := repository.NewUserInviteRepository(dbPool)
	issueReportRepository := repository.NewIssueReportRepository(dbPool)

	// Initialize app dependencies
	appServices := &app.Services{
		KV:     kv,
		Config: config,
		Logger: logger,
	}

	// The KV is in-process, so a start has no block flags: put every blocked
	// login's back before the first request (B034, user_block_handler.go).
	blockedUserNameList, err := userRepo.RetrieveBlockedUserNameList(
		context.Background())
	if err == nil {
		err = handler.LoadBlockedUsers(appServices, blockedUserNameList)
	}
	if err != nil {
		logger.Error("BLK_LOAD", "Failed to load blocked users: "+err.Error(),
			"username", "app-root")
		os.Exit(1)
	}
	logger.Info("BLK_LOAD", fmt.Sprintf("%d blocked login(s) flagged",
		len(blockedUserNameList)), "username", "app-root")

	// Initialize handlers
	healthHandler := handler.HealthHandler(appServices)
	authHandler := handler.AuthHandler(appServices, userRepo)
	refreshHandler := handler.RefreshHandler(appServices, userRepo)
	changeSecurityContextHandler := handler.ChangeSecurityContextHandler(
		appServices,
		userRepo,
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
	// The help module (doc/151-help-module.md): the routes answer 404
	// unless HELP_ENABLED=true.
	helpHandler := handler.NewHelpHandler(appServices, helpEventRepository)
	profileHandler := handler.NewProfileHandler(appServices, userRepo)
	userSelfCreateHandler := handler.UserSelfCreateHandler(
		appServices,
		userRepo,
	)
	userInviteHandler := handler.NewUserInviteHandler(
		appServices,
		appUserGroupRepo,
		userInviteRepository,
	)
	// The issue-report module (doc/150-issue-report-module.md): the routes
	// answer 404 unless ISSUE_REPORTS_ENABLED=true; the relay runs only
	// with its three target keys set.
	var issueRelayClient *issuerelay.Client
	var issueRelayDownloader handler.IssueRelayDownloader
	if config.IssueRelayEnabled() {
		issueRelayClient = issuerelay.NewClient(issuerelay.Config{
			BaseURL:   config.IssueRelayURL,
			ProjectID: config.IssueRelayProjectID,
			Token:     config.IssueRelayToken,
		}, nil)
		issueRelayDownloader = issueRelayClient
	}
	issueReportHandler := handler.NewIssueReportHandler(
		appServices,
		issueReportRepository,
		userRepo,
		issueRelayDownloader,
	)
	mfaSetupHandler := handler.MFASetupHandler(appServices, userRepo)
	mfaVerifyHandler := handler.MFAVerifyHandler(appServices, userRepo)

	// Setup routes & map handlers
	mux := http.NewServeMux()

	// Apply security headers middleware to all routes
	securedMux := security.HeadersMiddleware(mux)

	// Apply CORS middleware if origins are configured
	if len(config.CorsAllowedOrigins) > 0 {
		securedMux = security.CORSMiddleware(
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
	mux.HandleFunc("GET /me", profileHandler.RetrieveOwnProfile)
	mux.HandleFunc("PUT /me", profileHandler.UpdateOwnProfile)
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

	// Invite links (public half): read and redeem a link, no session
	// (user_invite_handler.go)
	mux.HandleFunc("POST /inviteCheck", userInviteHandler.CheckInvite)
	mux.HandleFunc("POST /inviteRedeem", userInviteHandler.RedeemInvite)

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

	// Invite links (admin half): an invited login and its one-time link
	mux.HandleFunc("POST /userInvite", userInviteHandler.CreateInvitedUser)
	mux.HandleFunc("POST /userInviteReissue",
		userInviteHandler.ReissueInvite)
	mux.HandleFunc("POST /userInviteRevoke", userInviteHandler.RevokeInvite)

	// Admin reset MFA
	adminResetMFAHandler := handler.AdminResetUserMFAHandler(
		appServices,
		userRepo,
	)
	mux.HandleFunc("POST /admin/resetUserMFA", adminResetMFAHandler)

	// Block / unblock a login: account_status for /auth, revocation and the
	// BLOCKED* KV flag for live sessions (user_block_handler.go)
	mux.HandleFunc("POST /admin/blockUser",
		handler.AdminBlockUserHandler(appServices, userRepo, true))
	mux.HandleFunc("POST /admin/unblockUser",
		handler.AdminBlockUserHandler(appServices, userRepo, false))

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

	// Client error reports (signed-in callers only)
	mux.HandleFunc("POST /error-report", errorReportHandler.HandleErrorReport)

	// In-app issue reports (issue_report_handler.go): anyone signed in;
	// the tenant's reports
	mux.HandleFunc("POST /issueReport", issueReportHandler.CreateIssueReport)
	mux.HandleFunc("GET /issueReport",
		issueReportHandler.RetrieveIssueReport)
	mux.HandleFunc("GET /issueReportList",
		issueReportHandler.RetrieveIssueReportList)
	mux.HandleFunc("POST /issueReportMessage",
		issueReportHandler.CreateIssueReportMessage)
	mux.HandleFunc("POST /issueReportAction",
		issueReportHandler.CreateIssueReportAction)
	mux.HandleFunc("GET /issueReportAttachment",
		issueReportHandler.RetrieveIssueReportAttachment)
	// In-app help usage (help_handler.go): anyone signed in records; a
	// user admin reads the tenant's counts
	mux.HandleFunc("POST /helpEvent", helpHandler.CreateHelpEvent)
	mux.HandleFunc("GET /helpInsights", helpHandler.RetrieveHelpInsights)

	// Create server. JoinHostPort brackets an IPv6 bind ("[::1]:8080"). The
	// address is never empty: LoadConfig requires SERVER_BIND_ADDRESS.
	server := newHTTPServer(
		net.JoinHostPort(
			config.ServerBindAddress, strconv.Itoa(config.ServerPort),
		),
		securedMux,
	)

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

	// The issue relay: sends stored reports to the tracker and pulls back
	// status and replies, until shutdown.
	relayContext, relayCancel := context.WithCancel(context.Background())
	defer relayCancel()
	issueModuleState := "off (ISSUE_REPORTS_ENABLED is not true)"
	if config.IssueReportsEnabled {
		issueModuleState = "on"
	}
	logger.Info("ISSUE_MODULE", "Issue reports "+issueModuleState,
		"username", "app-root")
	if issueRelayClient != nil {
		relay := &issuerelay.Relay{
			Client: issueRelayClient,
			Store:  issueReportRepository,
			Log:    logger,
		}
		logger.Info("ISSUE_RELAY_MODE", fmt.Sprintf(
			"Issue relay on, every %ds", config.IssueRelayIntervalSeconds),
			"username", "app-root")
		go relay.Run(relayContext, time.Duration(
			config.IssueRelayIntervalSeconds)*time.Second)
	} else {
		logger.Info("ISSUE_RELAY_MODE", "Issue relay off "+
			"(ISSUE_RELAY_URL, ISSUE_RELAY_PROJECT_ID and "+
			"ISSUE_RELAY_TOKEN not set); reports stay in the app",
			"username", "app-root")
	}
	helpModuleState := "off (HELP_ENABLED is not true)"
	if config.HelpEnabled {
		helpModuleState = "on"
	}
	logger.Info("HELP_MODULE", "Help "+helpModuleState,
		"username", "app-root")

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

// newHTTPServer builds the server with its timeouts (GRC RG0043, C015).
// Without them a client that trickles headers or a body holds a connection
// open forever, and nginx was the only guard (none in local dev, or where a
// derived app runs without it). The header has 10 s; the whole request
// (nginx's 20M client_max_body_size on a slow link) 2 min; the response
// 5 min, nginx's proxy_read_timeout in infra/*/etc/nginx_*, so a long
// download or render is cut by neither side first; an idle keep-alive
// 2 min. A route that needs longer raises its own deadline with
// http.ResponseController.
func newHTTPServer(bindAddress string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              bindAddress,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}
}
