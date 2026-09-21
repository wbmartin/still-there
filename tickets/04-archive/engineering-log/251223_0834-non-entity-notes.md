> Archived from `doc/archive/811-non-entity-notes.md` on 2026-09-21 by PS0006; point-in-time, not maintained.

# Non-Entity Application Code Notes

This document contains notes about maintaining and updating non-entity related application code components.

## Client Updates
When shrmpl-srv-preview updates KV or logging clients, copy the improved versions from `shrmpl-srv-preview/examples/go/shrmpl/` to `src/api-srv/internal/app_lib/`, rename to `kv_client.go` and `logging_client.go`, change package to `app_lib`, and adapt function signatures to eliminate config dependencies (e.g., `NewKV(kvHostPort string)`, `NewLogger(serverName, logReceiverHostPort string)`). Update main.go calls accordingly and restart services.