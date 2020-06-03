BEGIN;

ALTER TABLE channels ADD COLUMN last_seen timestamptz DEFAULT NOW() NOT NULL;

COMMIT;
