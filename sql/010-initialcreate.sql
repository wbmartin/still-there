-- Initial database schema for the starter API server
-- This creates the core tables for authentication, authorization, and the example example entity

-- User grants table (system-wide, security_context = 0 for system grants)
CREATE TABLE user_grant (
    user_grant_id INTEGER PRIMARY KEY,
    grant_short_code VARCHAR(5) NOT NULL UNIQUE,
    grant_description TEXT,
    security_context INTEGER NOT NULL,
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);

-- User groups table (per security context)
CREATE TABLE user_group (
    user_group_id SERIAL PRIMARY KEY,
    group_name VARCHAR(255) NOT NULL,
    user_grant_list INTEGER[] NOT NULL,
    security_context INTEGER NOT NULL,
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER,
    UNIQUE (group_name, security_context)
);

-- Application users table
CREATE TABLE app_user (
    app_user_id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    pw_hash VARCHAR(125) NOT NULL,
    first_name VARCHAR(200) NOT NULL,
    last_name VARCHAR(200) NOT NULL,
    email VARCHAR(200) NOT NULL,
    security_context INTEGER NOT NULL DEFAULT 1,
    user_group_membership_list INTEGER[],
    mfa_exempt BOOLEAN NOT NULL DEFAULT FALSE,
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);

-- Example example entity table
CREATE TABLE example(
    example_id SERIAL PRIMARY KEY,
    example_text TEXT NOT NULL,
    example_int INTEGER,
    example_float FLOAT,
    example_int_array INTEGER[],
    example_float_array FLOAT[],
    security_context INTEGER NOT NULL,
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);

-- Indexes for performance
CREATE INDEX idx_app_user_username ON app_user(username);
CREATE INDEX idx_app_user_security_context ON app_user(security_context);
CREATE INDEX idx_user_group_security_context ON user_group(security_context);
CREATE INDEX idx_example_security_context ON example(security_context);
CREATE INDEX idx_example_delete_user_id ON example(delete_user_id);
