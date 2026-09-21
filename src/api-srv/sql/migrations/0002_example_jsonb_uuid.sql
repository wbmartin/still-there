-- PS0016: the example entity exercises jsonb and uuid so a derived entity has
-- a worked pattern for both. example_uuid is set in Go on create and never
-- changes; the backfill below is the one place the database generates it.
-- example_jsonb holds any JSON value and may be NULL.

ALTER TABLE example
    ADD COLUMN example_uuid UUID,
    ADD COLUMN example_jsonb JSONB;

UPDATE example SET example_uuid = gen_random_uuid() WHERE example_uuid IS NULL;

ALTER TABLE example ALTER COLUMN example_uuid SET NOT NULL;

CREATE UNIQUE INDEX idx_example_uuid ON example(example_uuid);
