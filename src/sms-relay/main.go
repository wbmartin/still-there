// sms-relay takes Telnyx's inbound-text webhook for several apps, checks
// the signature once, stores each text in its own Postgres table under
// the queue the URL names, and lets each app read its queue over HTTP:
// peek, receive (lease), ack. Delivery is at least once; an app
// de-duplicates by message id. README.md has the API.
//
//	sms-relay serve <env file>     the webhook and the consumer API
//	sms-relay migrate <env file>   apply the embedded migrations (DDL role)
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const usage = "usage: sms-relay serve|migrate <env file>"

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = serve(os.Args[2])
	case "migrate":
		err = migrate(os.Args[2])
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "sms-relay:", err)
		os.Exit(1)
	}
}

func migrate(path string) error {
	databaseURL, err := loadDatabaseURL(path)
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	count, err := migrateUp(ctx, db, epochNow())
	if err != nil {
		return err
	}
	fmt.Printf("migrate: %d applied; schema current\n", count)
	return nil
}

func serve(path string) error {
	config, err := loadConfig(path)
	if err != nil {
		return fmt.Errorf("configuration: %w", err)
	}
	log := slog.New(slog.NewJSONHandler(os.Stderr,
		&slog.HandlerOptions{Level: config.LogLevel}))
	ctx := context.Background()
	db, err := pgxpool.New(ctx, config.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("database: %w", err)
	}
	if err := checkSchema(ctx, db); err != nil {
		return err
	}
	publicKey, _ := parsePublicKey(config.TelnyxPublicKey) // checked by loadConfig
	store := &Store{db: db}
	relay := &Server{
		Store: store, PublicKey: publicKey,
		TokenList: config.ConsumerTokenList, Log: log,
	}
	server := &http.Server{
		Addr: net.JoinHostPort(config.ServerBindAddress,
			strconv.Itoa(config.ServerPort)),
		Handler:           relay.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	retainContext, retainCancel := context.WithCancel(ctx)
	defer retainCancel()
	retainDone := make(chan struct{})
	go func() {
		defer close(retainDone)
		retain(retainContext, store, config.RetainDays, log)
	}()

	serveErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", server.Addr,
			"queueTokenCount", len(config.ConsumerTokenList))
		serveErr <- server.ListenAndServe()
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serveErr:
		return err
	case <-quit:
	}
	log.Info("shutting down")
	shutdownContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err = server.Shutdown(shutdownContext)
	retainCancel()
	<-retainDone
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("stopped")
	return nil
}

// retain deletes messages acked more than retainDays ago, at start and
// then daily, until ctx is done.
func retain(ctx context.Context, store *Store, retainDays int, log *slog.Logger) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		cutoff := epochNow() - float64(retainDays*86400)
		if count, err := store.DeleteDoneBefore(ctx, cutoff); err != nil {
			log.Error("retention", "err", err)
		} else if count > 0 {
			log.Info("retention: deleted", "count", count)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
