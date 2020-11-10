BEGIN;

ALTER TABLE channels DROP COLUMN filter_exempt_level;

COMMIT;
