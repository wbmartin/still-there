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
INSERT INTO app_user (username, pw_hash, first_name, last_name, email, security_context, user_group_membership_list, mfa_exempt, create_ts, create_user_id, update_ts, update_user_id)
SELECT
    'admin',
    '$2a$10$8K1p/5w6Yg3B8rXcLpVzAe9dQwJcKvJcKvJcKvJcKvJcKvJcKvJcKv', -- bcrypt hash for 'admin123!'
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