-- An inbound text read from the SMS relay (src/sms-relay) records the
-- relay's message id on its sms_message row. The relay delivers at least
-- once, so a redelivered id already recorded is skipped; the unique index
-- keeps one row per id. NULL for texts from /smsInbound, the simulate
-- route and every outbound text.
ALTER TABLE sms_message ADD COLUMN relay_message_id BIGINT;
CREATE UNIQUE INDEX idx_sms_message_relay_message_id
    ON sms_message(relay_message_id) WHERE relay_message_id IS NOT NULL;
