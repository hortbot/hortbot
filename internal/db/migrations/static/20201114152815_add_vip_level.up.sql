-- https://stackoverflow.com/a/10404041

BEGIN;

ALTER TYPE access_level RENAME TO access_level__;

CREATE TYPE access_level AS ENUM (
    'everyone',
    'subscriber',
    'vip',
    'moderator',
    'broadcaster',
    'admin'
);

ALTER TABLE channels
    ALTER COLUMN mode TYPE access_level USING mode::text::access_level,
    ALTER COLUMN roll_level TYPE access_level USING roll_level::text::access_level,
    ALTER COLUMN filter_exempt_level DROP DEFAULT, -- No longer needed; previous migration added "subscriber" to existing columns and it's explicitly inserted later as with other columns.
    ALTER COLUMN filter_exempt_level TYPE access_level USING filter_exempt_level::text::access_level;

ALTER TABLE command_infos
    ALTER COLUMN access_level TYPE access_level USING access_level::text::access_level;

DROP TYPE access_level__;

COMMIT;
