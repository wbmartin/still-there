-- Initial seed data for the starter API server
-- This includes default user grants, user groups, and an admin user

WITH common_vars AS (
    SELECT
        EXTRACT(EPOCH FROM NOW()) as ts,
        0 as system_user_id
)


-- Insert user groups for security context 1
INSERT INTO user_group (group_name, user_grant_list, security_context, create_ts, create_user_id, update_ts, update_user_id)
SELECT
    data.group_name,
    data.user_grant_list,
    1,
    common_vars.ts,
    common_vars.system_user_id,
    common_vars.ts,
    common_vars.system_user_id
FROM common_vars,
(VALUES
    ('Basic User', ARRAY[1001, 1002, 1003, 1004, 8001, 9010], 1),
    ('Admin', ARRAY[1001, 1002, 1003, 1004, 8001, 9001, 9002, 9003, 9010, 9011, 9012, 9013], 1)
) AS data(group_name, user_grant_list, security_context);

-- Insert admin user (password hash for 'admin123!')
INSERT INTO app_user (user_name, pw_hash, first_name, last_name, email, security_context, user_group_membership_list, mfa_exempt, create_ts, create_user_id, update_ts, update_user_id)
SELECT
    'admin',
    '$argon2id$v=19$m=65536,t=3,p=4$dvj9Tc0FPrJvKevg+DCOhg$TlKP/tTUBnTZ0VIR8IU4JAYQl8i1jnCcIRUrPgWCUIg' -- Argon2id hash for 'admin123!' change for non-dev/server builds
    'System',
    'Administrator',
    'admin@example.com',
    1,
    ARRAY[2], -- Admin group
    TRUE,    -- MFA exempt for initial setup
    common_vars.ts,
    common_vars.system_user_id,
    common_vars.ts,
    common_vars.system_user_id
FROM common_vars;

-- Insert example data
INSERT INTO example (example_text, example_int, example_float, example_int_array, example_float_array, security_context, create_ts, create_user_id, update_ts, update_user_id)
SELECT
    data.example_text,
    data.example_int,
    data.example_float,
    data.example_int_array,
    data.example_float_array,
    1,
    common_vars.ts,
    common_vars.system_user_id,
    common_vars.ts,
    common_vars.system_user_id
FROM common_vars,
(VALUES
    (1, 'Sample example 1', 42, 3.14, ARRAY[1,2,3], ARRAY[1.1,2.2,3.3]),
    (2, 'Sample example 2', NULL, NULL, NULL, NULL)
) AS data(example_id, example_text, example_int, example_float, example_int_array, example_float_array);

-- Reset the sequence to the next available ID
SELECT setval('example_example_id_seq', (SELECT COALESCE(MAX(example_id), 0) + 1 FROM example));
