package app_lib

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wbmartin/starter-api/internal/repository"
)

// AppServices holds all the shared dependencies for handlers
type AppServices struct {
	DBPool        *pgxpool.Pool
	KV            ThisAppKVInterface
	Config        *Config
	Logger        ThisAppLoggerInterface
	UserRepo      *repository.UserRepository
	UserGroupRepo *repository.UserGroupRepository
	ExampleRepo   *repository.ExampleRepository
}
