-- B034: app_user.account_status says whether a login may be used: "active",
-- or "blocked" by an admin (POST /admin/blockUser), which /auth refuses.
-- Every login that exists today is in use, so the backfill is "active";
-- then the column is NOT NULL with no DEFAULT (doc/900: the Go insert names
-- the value). B035's invite links add a third value, "invited".

ALTER TABLE app_user ADD COLUMN account_status TEXT;

UPDATE app_user SET account_status = 'active' WHERE account_status IS NULL;

ALTER TABLE app_user ALTER COLUMN account_status SET NOT NULL;
