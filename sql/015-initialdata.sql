-- Initial seed data for the starter API server
-- This includes default user grants, user groups, and an admin user

WITH common_vars AS (
    SELECT
        EXTRACT(EPOCH FROM NOW()) as ts,
        0 as system_user_id
)
-- Insert user grants
INSERT INTO user_grant (user_grant_id, grant_short_code, grant_description, security_context, create_ts, create_user_id, update_ts, update_user_id)
SELECT
    data.user_grant_id,
    data.grant_short_code,
    data.grant_description,
    0,
    common_vars.ts,
    common_vars.system_user_id,
    common_vars.ts,
    common_vars.system_user_id
FROM common_vars,
(VALUES
    (1001, 'GET_EXAMPLE', 'Retrieve examples'),
    (1002, 'ADD_EXAMPLE', 'Add new examples'),
    (1003, 'CHG_EXAMPLE', 'Change existing examples'),
    (1004, 'DEL_EXAMPLE', 'Delete examples'),
    (8001, 'CHG_OWN_PW', 'Change own password'),
    (9001, 'ADD_USER', 'Add new users'),
    (9002, 'BLOCK_USER', 'Block user access'),
    (9003, 'CHG_OTHER_USER_PW', 'Reset another user password')
) AS data(user_grant_id, grant_short_code, grant_description);

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
    ('Basic User', ARRAY[1001, 1002, 1003, 1004, 8001], 1),
    ('Admin', ARRAY[1001, 1002, 1003, 1004, 8001, 9001, 9002, 9003], 1)
) AS data(group_name, user_grant_list, security_context);

-- Insert admin user (password hash for 'admin123!')
INSERT INTO app_user (user_name, pw_hash, first_name, last_name, email, security_context, user_group_membership_list, mfa_exempt, create_ts, create_user_id, update_ts, update_user_id)
SELECT
    'admin',
    '$2a$10$nrFkDg5BxLgTDxsFzKh2b.a/2ipwowdcey.nY/Si.7h7m8cm8Z0H.' -- bcrypt hash for 'admin123!' change for non-dev builds
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
