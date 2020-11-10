BEGIN;

ALTER TABLE channels ADD COLUMN filter_exempt_level access_level DEFAULT 'subscriber' NOT NULL;

COMMIT;
