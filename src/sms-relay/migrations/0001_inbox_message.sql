-- One row per inbound text the relay accepted, for the app whose queue
-- the webhook URL named. Timestamps are epoch seconds, set in Go.
-- A consumer's receive leases rows (lease_id, lease_until_ts); its ack
-- sets done_ts. An expired lease makes the row receivable again, so
-- delivery is at least once and receive_count says how often.
CREATE TABLE inbox_message (
    inbox_message_id BIGSERIAL PRIMARY KEY,
    queue_name TEXT NOT NULL,
    received_ts NUMERIC(17,7) NOT NULL,
    provider TEXT NOT NULL,
    event_type TEXT NOT NULL,
    from_phone TEXT NOT NULL,
    to_phone TEXT NOT NULL,
    body_text TEXT NOT NULL,
    raw_body JSONB NOT NULL,
    lease_id TEXT,
    lease_until_ts NUMERIC(17,7),
    receive_count INTEGER NOT NULL,
    done_ts NUMERIC(17,7)
);
CREATE INDEX idx_inbox_message_queue
    ON inbox_message(queue_name, done_ts, lease_until_ts, inbox_message_id);
