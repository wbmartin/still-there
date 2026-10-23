package app

import (
	"stillthere-api/internal/cache"
	"stillthere-api/internal/config"
	"stillthere-api/internal/logging"
)

// Services holds the shared dependencies every handler gets: the
// config, the in-process store and the logger. Repositories are passed to
// the handlers that need them; nothing reaches the pool directly.
type Services struct {
	KV     *cache.KV
	Config *config.Config
	Logger *logging.Logger
}
