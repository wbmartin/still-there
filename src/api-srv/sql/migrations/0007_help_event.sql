-- The help module (doc/151-help-module.md): in-app help usage, so an
-- admin can see where the help falls short (GET /helpInsights). The web
-- app's Help panel and page tours post one row per event, best-effort;
-- nothing in the app reads a single row back.
--
-- The table exists in every app built from the template; the module is
-- dormant (no route answers) until HELP_ENABLED=true.
--
-- 0007 follows the issue-report module's 0006, on which this change is
-- stacked.
--
-- event: search_no_result (query = what was typed, cut to 200 characters;
-- entry_id NULL) | entry_view (entry_id = a glossary term or topic id) |
-- tour_complete | tour_skip (entry_id = the tour's id). route is the web
-- app route the panel was open on. app_user_id is the caller, kept for the
-- per-user daily cap; the insights view counts by key, never by person.
-- Rows are never updated: update_ts / update_user_id repeat the create
-- pair. query is free text a user typed and may hold anything they typed.
CREATE TABLE help_event (
    help_event_id SERIAL PRIMARY KEY,
    security_context_id INTEGER NOT NULL,
    app_user_id INTEGER NOT NULL,
    event VARCHAR(24) NOT NULL,
    entry_id VARCHAR(64),
    route VARCHAR(64) NOT NULL,
    query VARCHAR(200),
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);

-- The insights read: one tenant's events of one kind since a time.
CREATE INDEX idx_help_event_sc_event_create_ts
    ON help_event(security_context_id, event, create_ts);
-- The per-user daily cap.
CREATE INDEX idx_help_event_sc_app_user_id_create_ts
    ON help_event(security_context_id, app_user_id, create_ts);

-- A new table needs its own grants (doc/900).
GRANT SELECT ON help_event TO db_reader;
GRANT ALL PRIVILEGES ON help_event TO db_writer;
GRANT ALL ON SEQUENCE help_event_help_event_id_seq TO db_writer;
