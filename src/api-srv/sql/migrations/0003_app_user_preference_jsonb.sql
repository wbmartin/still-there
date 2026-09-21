-- PS0018: per-user preferences (colour scheme, rows per page, ...) as one
-- jsonb document the user edits through PUT /me. NULL until the user saves
-- something; the web app applies defaults for any key that is absent.

ALTER TABLE app_user ADD COLUMN preference_jsonb JSONB;
