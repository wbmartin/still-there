-- Tenant 1 does not exist in a test database. Run on a freshly migrated and
-- seeded SCRATCH database (bin/510-reset-test-db, after test-seed.sql);
-- never on a developer's or a deployed database, and never a migration.
--
-- Puts CHECK (security_context_id <> 1) on every table that carries
-- security_context_id, read from the catalog so a new table is covered with
-- no edit here. A test that writes tenant 1 then fails at the database
-- instead of passing quietly. TRUNCATE (bin/515-reseed-db) keeps the
-- constraint, so a reseed can never bring tenant 1 back. Idempotent: a
-- table that already has the constraint is skipped, so bin/515 re-runs it
-- to cover a database built before a new table existed.
DO $$
DECLARE
    tenant_table record;
    tenant_one_count bigint;
BEGIN
    FOR tenant_table IN
        SELECT c.table_name
        FROM information_schema.columns c
        JOIN information_schema.tables t USING (table_schema, table_name)
        WHERE c.table_schema = 'public'
          AND c.column_name = 'security_context_id'
          AND t.table_type = 'BASE TABLE'
    LOOP
        EXECUTE format(
            'SELECT count(*) FROM public.%I WHERE security_context_id = 1',
            tenant_table.table_name) INTO tenant_one_count;
        IF tenant_one_count > 0 THEN
            RAISE EXCEPTION '% still holds % tenant-1 row(s)',
                tenant_table.table_name, tenant_one_count;
        END IF;
        CONTINUE WHEN EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conname = 'test_db_never_tenant_1'
              AND conrelid = format('public.%I',
                  tenant_table.table_name)::regclass);
        EXECUTE format(
            'ALTER TABLE public.%I ADD CONSTRAINT test_db_never_tenant_1 '
            'CHECK (security_context_id <> 1)',
            tenant_table.table_name);
    END LOOP;
END $$;
