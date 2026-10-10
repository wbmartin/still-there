-- The issue-report module (doc/150-issue-report-module.md): a signed-in
-- user files a bug, polish or feature report from inside the app, with a
-- markdown description and up to five files; everyone in the tenant sees
-- the tenant's reports and can add to their conversation. Stored here
-- first, so nothing is lost while the tracker the reports go to is
-- unreachable or not configured; the issue relay (internal/issuerelay)
-- sends them on and pulls back status and staff replies.
--
-- The tables exist in every app built from the template; the module is
-- dormant (no route answers, no relay runs) until the config turns it on.
--
-- issue_report
--   kind: bug | polish | feature.
--   state: the tracker's issue state (open, routed, fixing, reverify,
--   closed, reopened, wont_fix); status is its plain-language status
--   ("Under review", "Being fixed", ...), status_note its note.
--   reporter_name / reporter_email: copied from the filer's login when the
--   report is made, so a later rename or a switch of tenant never changes
--   who filed it, and the relay reads no other table.
--   context_jsonb: what the browser sends to help reproduce (page,
--   browser, viewport); any JSON object or NULL.
--   remote_id: the tracker's id once sent. relay_state: pending (to send)
--   | sent | failed (last try failed; retried until relay_attempts reaches
--   the relay's cap) | local (no relay configured when it was filed).
CREATE TABLE issue_report (
    issue_report_id SERIAL PRIMARY KEY,
    security_context_id INTEGER NOT NULL,
    kind VARCHAR(16) NOT NULL,
    summary VARCHAR(500) NOT NULL,
    detail TEXT,
    page_url VARCHAR(1000),
    context_jsonb JSONB,
    state VARCHAR(16) NOT NULL,
    status VARCHAR(120) NOT NULL,
    status_note TEXT,
    reporter_name VARCHAR(200) NOT NULL,
    reporter_email VARCHAR(200) NOT NULL,
    remote_id VARCHAR(64),
    relay_state VARCHAR(16) NOT NULL,
    relay_error VARCHAR(500),
    relay_attempts INTEGER NOT NULL,
    last_synced_ts NUMERIC(17,7),
    last_activity_ts NUMERIC(17,7) NOT NULL,
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);
CREATE INDEX idx_issue_report_sc_last_activity_ts
    ON issue_report(security_context_id, last_activity_ts);
CREATE INDEX idx_issue_report_relay_state ON issue_report(relay_state);
CREATE UNIQUE INDEX idx_issue_report_remote_id
    ON issue_report(remote_id) WHERE remote_id IS NOT NULL;

-- One entry in a report's conversation. author_kind: reporter (a user of
-- this app) | staff (a tracker reply, pulled by the relay). action: NULL,
-- withdraw or reopen (a reporter's action, relayed as one). remote_id
-- keeps a staff reply from being stored twice. relay_state as on
-- issue_report, plus none (a staff reply: nothing to send).
CREATE TABLE issue_report_message (
    issue_report_message_id SERIAL PRIMARY KEY,
    security_context_id INTEGER NOT NULL,
    issue_report_id INTEGER NOT NULL,
    author_kind VARCHAR(16) NOT NULL,
    author_name VARCHAR(200),
    body TEXT NOT NULL,
    action VARCHAR(16),
    remote_id VARCHAR(64),
    relay_state VARCHAR(16) NOT NULL,
    relay_error VARCHAR(500),
    relay_attempts INTEGER NOT NULL,
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);
CREATE INDEX idx_issue_report_message_sc_report
    ON issue_report_message(security_context_id, issue_report_id);
CREATE INDEX idx_issue_report_message_relay_state
    ON issue_report_message(relay_state);
CREATE UNIQUE INDEX idx_issue_report_message_remote_id
    ON issue_report_message(issue_report_id, remote_id)
    WHERE remote_id IS NOT NULL;

-- A file on a report, or on one of its messages (issue_report_message_id
-- NULL: on the report itself). The bytes are stored here (bytea).
-- content_type is the type the server read from the bytes, never one a
-- client sent. A staff file pulled from the tracker has content NULL and
-- remote_id set; it is fetched through the relay when downloaded.
CREATE TABLE issue_report_attachment (
    issue_report_attachment_id SERIAL PRIMARY KEY,
    security_context_id INTEGER NOT NULL,
    issue_report_id INTEGER NOT NULL,
    issue_report_message_id INTEGER,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size_bytes INTEGER NOT NULL,
    content BYTEA,
    remote_id VARCHAR(64),
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);
CREATE INDEX idx_issue_report_attachment_sc_report
    ON issue_report_attachment(security_context_id, issue_report_id);
CREATE UNIQUE INDEX idx_issue_report_attachment_remote_id
    ON issue_report_attachment(issue_report_id, remote_id)
    WHERE remote_id IS NOT NULL;

-- 0001's grants covered the tables that existed then; a new table needs its
-- own (doc/900).
GRANT SELECT ON issue_report, issue_report_message, issue_report_attachment
    TO db_reader;
GRANT ALL PRIVILEGES
    ON issue_report, issue_report_message, issue_report_attachment
    TO db_writer;
GRANT ALL ON SEQUENCE issue_report_issue_report_id_seq,
    issue_report_message_issue_report_message_id_seq,
    issue_report_attachment_issue_report_attachment_id_seq
    TO db_writer;
