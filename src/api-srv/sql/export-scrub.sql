-- The credential scrub bin/620-export-tenant-data runs, as one transaction,
-- on its throwaway copy of the local database before dumping it for a
-- post-sales host (doc/330 0.4). It changes credentials only, never a data
-- row, and REFUSES, naming what it found, data it must not carry:
--
--   * any row in tenant 2, the test tenant (CLAUDE.md): test data is never
--     a customer's, and no real tenant is numbered 2
--   * a live fixture login -- admin, basic, mfa, switcher, the names
--     sql/fixtures/local-seed.sql and test-seed.sql seed, whose passwords
--     are in the repo. Rename it or delete it in the source database first.
--     Keep this list in step with the seeds.
--
-- Then no login keeps a credential that lived on the laptop. Every pw_hash
-- and totp_seed is emptied ('' never verifies; a seed would not decrypt
-- under the host's TOTP_ENCRYPTION_KEY anyway), mfa_setup and mfa_exempt are
-- cleared, and every live login that is not blocked becomes "invited"
-- (B035): the first person in on the host signs in through an invite link
-- issued there (doc/330 A9), sets a password and enrols in MFA. A blocked
-- login stays blocked; a deleted one keeps its status but loses its
-- credentials too. Open invite links are revoked: their tokens were shown
-- on the laptop.
--
-- Plain SQL, no psql meta-commands: sql/export_scrub_test.go runs this file
-- as written against a scratch database.

DO $$
DECLARE
    tenant_table record;
    row_count bigint;
    found_list text := '';
BEGIN
    FOR tenant_table IN
        SELECT c.table_name
        FROM information_schema.columns c
        JOIN information_schema.tables t USING (table_schema, table_name)
        WHERE c.table_schema = 'public'
          AND c.column_name = 'security_context_id'
          AND t.table_type = 'BASE TABLE'
        ORDER BY c.table_name
    LOOP
        EXECUTE format(
            'SELECT count(*) FROM public.%I WHERE security_context_id = 2',
            tenant_table.table_name) INTO row_count;
        IF row_count > 0 THEN
            found_list := found_list
                || format(' %s(%s)', tenant_table.table_name, row_count);
        END IF;
    END LOOP;
    IF found_list <> '' THEN
        RAISE EXCEPTION 'export refused: rows in tenant 2, the test '
            'tenant:%', found_list;
    END IF;

    SELECT string_agg(user_name, ', ' ORDER BY user_name) INTO found_list
    FROM app_user
    WHERE delete_ts IS NULL
      AND user_name IN ('admin', 'basic', 'mfa', 'switcher');
    IF found_list IS NOT NULL THEN
        RAISE EXCEPTION 'export refused: fixture logins present (their '
            'passwords are in the repo; rename or delete them first): %',
            found_list;
    END IF;
END $$;

UPDATE user_invite
SET revoked_ts = extract(epoch FROM now()),
    update_ts = extract(epoch FROM now()),
    update_user_id = 0
WHERE redeemed_ts IS NULL AND revoked_ts IS NULL AND delete_ts IS NULL;

UPDATE app_user
SET pw_hash = '',
    totp_seed = '',
    mfa_setup = FALSE,
    mfa_exempt = FALSE,
    account_status = CASE
        WHEN delete_ts IS NULL AND account_status <> 'blocked'
        THEN 'invited'
        ELSE account_status
    END,
    update_ts = extract(epoch FROM now()),
    update_user_id = 0;
