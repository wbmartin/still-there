-- Initial seed data for the starter API server
-- This includes default user grants, user groups, and an admin user

-- Insert user groups for security context 1
INSERT INTO app_user_group (group_name, user_grant_list, security_context, create_ts, create_user_id, update_ts, update_user_id)
VALUES
    ('Basic User', ARRAY[1001, 1002, 1003, 1004, 8001, 9010], 1, EXTRACT(EPOCH FROM NOW()), 0, EXTRACT(EPOCH FROM NOW()), 0),
    ('Admin', ARRAY[1001, 1002, 1003, 1004, 8001, 9001, 9002, 9003, 9010, 9011, 9012, 9013, 9014, 9015], 1, EXTRACT(EPOCH FROM NOW()), 0, EXTRACT(EPOCH FROM NOW()), 0);

-- Insert user groups for security context 2 (mirroring context 1)
INSERT INTO app_user_group (group_name, user_grant_list, security_context, create_ts, create_user_id, update_ts, update_user_id)
VALUES
    ('Basic2 User', ARRAY[1001, 1002, 1003, 1004, 8001, 9010], 2, EXTRACT(EPOCH FROM NOW()), 0, EXTRACT(EPOCH FROM NOW()), 0),
    ('Admin2', ARRAY[1001, 1002, 1003, 1004, 8001, 9001, 9002, 9003, 9010, 9011, 9012, 9013, 9014, 9015], 2, EXTRACT(EPOCH FROM NOW()), 0, EXTRACT(EPOCH FROM NOW()), 0);

-- Insert admin user (password hash for 'admin123!')
      INSERT INTO app_user (user_name, pw_hash, first_name, last_name, email, security_context, app_user_group_membership_list, mfa_exempt, create_ts, create_user_id, update_ts, update_user_id)
VALUES
    ('admin',
     '$argon2id$v=19$m=65536,t=3,p=4$dvj9Tc0FPrJvKevg+DCOhg$TlKP/tTUBnTZ0VIR8IU4JAYQl8i1jnCcIRUrPgWCUIg', -- Argon2id hash for 'admin123!' change for non-dev/server builds
     'System',
     'Administrator',
     'admin@example.com',
     1,
     ARRAY[2], -- Admin group
     TRUE,     -- MFA exempt for initial setup
     EXTRACT(EPOCH FROM NOW()),
     0,
     EXTRACT(EPOCH FROM NOW()),
     0);

INSERT INTO app_user (user_name, pw_hash, first_name, last_name, email, security_context, app_user_group_membership_list, mfa_exempt, create_ts, create_user_id, update_ts, update_user_id)
     VALUES
         ('basic',
          '$argon2id$v=19$m=65536,t=3,p=4$kxssq7+N6Of3OSiUThJgrw$S+uHFwjOhp8daiHCYhyto7whkrDjDr2TD0+Y6Cq1kd4', -- Argon2id hash for 'basic123!'
          'Basic',
          'User',
          'basic@example.com',
          1,
          ARRAY[1], -- Basic User group
          TRUE,    -- Not MFA exempt
          EXTRACT(EPOCH FROM NOW()),
          0,
          EXTRACT(EPOCH FROM NOW()),
          0);

-- Add admin user to group 7 for additional context access
UPDATE app_user SET app_user_group_membership_list = ARRAY[2, 7] WHERE app_user_id = 1;

-- Insert example data
WITH common_vars AS (
    SELECT EXTRACT(EPOCH FROM NOW()) as ts, 0 as system_user_id
)
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
