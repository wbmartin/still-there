-- B035: invite a user with a one-time link instead of an admin-set password.
--
-- app_user.account_status (0004) gains a third value, "invited": a login an
-- admin created with no password yet. It cannot sign in (/auth refuses
-- anything not active) until its holder redeems the link, which sets the
-- password and makes it "active". The column is TEXT with no CHECK, so the
-- value needs no DDL; it is named here so the schema history says where it
-- comes from.
--
-- user_invite is one link that lets its holder set the password of one
-- invited login. Only the hex SHA-256 of the token is stored: the token is
-- shown to the admin once and is never readable from here. A row is live
-- while redeemed_ts, revoked_ts and delete_ts are NULL and expires_ts is
-- ahead. revoked_ts is set by a revoke, or by issuing a newer link for the
-- same login. redeemed_source_ip is the client that redeemed it.

CREATE TABLE user_invite (
    user_invite_id SERIAL PRIMARY KEY,
    security_context_id INTEGER NOT NULL,
    app_user_id INTEGER NOT NULL,
    token_hash TEXT NOT NULL,
    expires_ts NUMERIC(17,7) NOT NULL,
    redeemed_ts NUMERIC(17,7),
    redeemed_source_ip TEXT,
    revoked_ts NUMERIC(17,7),
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);

-- The public routes look a row up by its token alone, before there is a
-- tenant to scope by.
CREATE UNIQUE INDEX idx_user_invite_token_hash ON user_invite(token_hash);
CREATE INDEX idx_user_invite_sc_app_user_id
    ON user_invite(security_context_id, app_user_id);
CREATE INDEX idx_user_invite_sc_update_ts
    ON user_invite(security_context_id, update_ts);

-- 0001's grants covered the tables that existed then; a new table needs its
-- own (doc/900).
GRANT SELECT ON user_invite TO db_reader;
GRANT ALL PRIVILEGES ON user_invite TO db_writer;
GRANT ALL ON SEQUENCE user_invite_user_invite_id_seq TO db_writer;
