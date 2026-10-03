-- Seed data for the SCRATCH databases the test suites use
-- (loc_<slug>_test_db and loc_<slug>_e2e_db): the fixture groups, an admin,
-- a basic, an MFA and a two-tenant user, and two example rows -- all in the
-- test tenants.
--
-- A FIXTURE, not a migration. bin/510-reset-test-db runs it after
-- `migrate up`, then sql/fixtures/test-tenant-lock.sql; bin/515-reseed-db
-- re-runs it after a TRUNCATE. The developer's own database takes
-- local-seed.sql (tenant 1) instead.
--
-- TENANTS (security_context_id): 1 is the first real tenant and never
-- appears in a test database -- the lock refuses it. 2 is the test tenant:
-- groups 1-3, the four users and the examples. 3 is the "other tenant"
-- the isolation tests aim at: groups 4 and 5. Group ids 1-5 are fixed; the
-- tests rely on them.
--
-- Idempotent: every INSERT skips rows that already exist.
--
-- Grant ids must match src/api-srv/internal/model (user grants). That code is
-- the source of truth; the arrays below are its projection into SQL.
--
-- Passwords: admin / admin123!, basic / basic123!, mfa / basic123!,
-- switcher / basic123! (Argon2id).

-- User groups for the test tenant, 2 (explicit ids: the users below refer
-- to them by id)
WITH common_vars AS (
    SELECT EXTRACT(EPOCH FROM NOW()) AS ts, 0 AS system_id
)
INSERT INTO app_user_group (
    app_user_group_id, group_name, user_grant_list, security_context_id,
    create_ts, create_user_id, update_ts, update_user_id
)
SELECT
    data.app_user_group_id,
    data.group_name,
    data.user_grant_list,
    2,
    common_vars.ts,
    common_vars.system_id,
    common_vars.ts,
    common_vars.system_id
FROM common_vars,
(VALUES
    (1, 'Basic User', ARRAY[1, 1001, 1002, 1003, 1004, 8001, 9010]),
    (2, 'Admin', ARRAY[1,1001,1002,1003,1004,8001,9001,9002,9009,9010,9011,9012,9013,9014,9015,9003,9004,9008,9020]),
    (3, 'New User', ARRAY[1, 1001])
) AS data(app_user_group_id, group_name, user_grant_list)
ON CONFLICT DO NOTHING;

-- User groups for the other tenant, 3 (mirroring the test tenant's)
WITH common_vars AS (
    SELECT EXTRACT(EPOCH FROM NOW()) AS ts, 0 AS system_id
)
INSERT INTO app_user_group (
    app_user_group_id, group_name, user_grant_list, security_context_id,
    create_ts, create_user_id, update_ts, update_user_id
)
SELECT
    data.app_user_group_id,
    data.group_name,
    data.user_grant_list,
    3,
    common_vars.ts,
    common_vars.system_id,
    common_vars.ts,
    common_vars.system_id
FROM common_vars,
(VALUES
    (4, 'Basic2 User', ARRAY[1, 1001, 1002, 1003, 1004, 8001, 9010]),
    (5, 'Admin2', ARRAY[1,1001,1002,1003,1004,8001,9001,9002,9009,9010,9011,9012,9013,9014,9015,9003,9004,9008,9020])
) AS data(app_user_group_id, group_name, user_grant_list)
ON CONFLICT DO NOTHING;

-- Explicit ids above bypass the sequence; move it past them.
SELECT setval('app_user_group_app_user_group_id_seq',
    (SELECT COALESCE(MAX(app_user_group_id), 0) + 1 FROM app_user_group));

-- Users, all in the test tenant (2): admin (Admin group) and basic (Basic
-- User group) are MFA exempt so the test suites can sign in with a password
-- alone; mfa (Basic User group, password basic123!) is NOT exempt and starts
-- with no seed, so the TOTP enrol → verify path is exercised end to end.
-- switcher (MFA exempt) is homed in 2 with Basic User there and Basic2 User
-- in 3, the one login that can /changeSecurityContext.
WITH common_vars AS (
    SELECT EXTRACT(EPOCH FROM NOW()) AS ts, 0 AS system_id
)
INSERT INTO app_user (
    user_name, pw_hash, first_name, last_name, email, security_context_id,
    app_user_group_membership_list, mfa_exempt, totp_seed, mfa_setup,
    create_ts, create_user_id, update_ts, update_user_id
)
SELECT
    data.user_name,
    data.pw_hash,
    data.first_name,
    data.last_name,
    data.email,
    2,
    data.app_user_group_membership_list,
    data.mfa_exempt,
    '',
    FALSE,
    common_vars.ts,
    common_vars.system_id,
    common_vars.ts,
    common_vars.system_id
FROM common_vars,
(VALUES
    ('admin',
     '$argon2id$v=19$m=65536,t=3,p=4$dvj9Tc0FPrJvKevg+DCOhg$TlKP/tTUBnTZ0VIR8IU4JAYQl8i1jnCcIRUrPgWCUIg',
     'System', 'Administrator', 'admin@example.com', ARRAY[2], TRUE),
    ('basic',
     '$argon2id$v=19$m=65536,t=3,p=4$kxssq7+N6Of3OSiUThJgrw$S+uHFwjOhp8daiHCYhyto7whkrDjDr2TD0+Y6Cq1kd4',
     'Basic', 'User', 'basic@example.com', ARRAY[1], TRUE),
    ('mfa',
     '$argon2id$v=19$m=65536,t=3,p=4$kxssq7+N6Of3OSiUThJgrw$S+uHFwjOhp8daiHCYhyto7whkrDjDr2TD0+Y6Cq1kd4',
     'Second', 'Factor', 'mfa@example.com', ARRAY[1], FALSE),
    ('switcher',
     '$argon2id$v=19$m=65536,t=3,p=4$kxssq7+N6Of3OSiUThJgrw$S+uHFwjOhp8daiHCYhyto7whkrDjDr2TD0+Y6Cq1kd4',
     'Context', 'Switcher', 'switcher@example.com', ARRAY[1, 4], TRUE)
) AS data(user_name, pw_hash, first_name, last_name, email,
          app_user_group_membership_list, mfa_exempt)
ON CONFLICT (user_name) DO NOTHING;

-- Example data (test tenant, 2). No unique key on example, so match on text.
WITH common_vars AS (
    SELECT EXTRACT(EPOCH FROM NOW()) AS ts, 0 AS system_id
)
INSERT INTO example (
    example_text, example_int, example_float, example_int_array,
    example_float_array, example_uuid, example_jsonb, security_context_id,
    create_ts, create_user_id, update_ts, update_user_id
)
SELECT
    data.example_text,
    data.example_int,
    data.example_float,
    data.example_int_array,
    data.example_float_array,
    gen_random_uuid(),
    data.example_jsonb,
    2,
    common_vars.ts,
    common_vars.system_id,
    common_vars.ts,
    common_vars.system_id
FROM common_vars,
(VALUES
    ('Sample example 1', 42, 3.14, ARRAY[1,2,3], ARRAY[1.1,2.2,3.3],
     '{"colour": "teal", "tagList": ["seed", "sample"]}'::jsonb),
    ('Sample example 2', NULL, NULL, NULL, NULL, NULL)
) AS data(example_text, example_int, example_float, example_int_array,
          example_float_array, example_jsonb)
WHERE NOT EXISTS (
    SELECT 1
    FROM example
    WHERE example.example_text = data.example_text
      AND example.security_context_id = 2
);
