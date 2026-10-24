-- The device monitoring grants (model/grants.go, migration 0008's tables)
-- for every existing admin group: a group holding RESET_MFA_TOKEN (9020),
-- which only the admin groups carry. A group that already has the device
-- grants is left alone. New databases get them from the seed fixtures.
-- update_ts is left as it was, as 0004 left it.
--
-- 1010-1014 device (+ ISSUE_DEVICE_TOKEN), 1020-1023 beacon,
-- 1030-1033 contact, 1040 / 1042 alert read / update,
-- 1050 / 1052 tenant setting read / update.
UPDATE app_user_group
SET user_grant_list = user_grant_list || ARRAY[
    1010, 1011, 1012, 1013, 1014, 1020, 1021, 1022, 1023,
    1030, 1031, 1032, 1033, 1040, 1042, 1050, 1052]
WHERE 9020 = ANY(user_grant_list)
  AND NOT (1010 = ANY(user_grant_list))
  AND delete_ts IS NULL;
