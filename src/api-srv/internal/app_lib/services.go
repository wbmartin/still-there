package app_lib

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// AppServices holds shared dependencies for handlers (excluding repositories)
type AppServices struct {
	DBPool *pgxpool.Pool
	KV     *KV
	Config *Config
	Logger *Logger
}
