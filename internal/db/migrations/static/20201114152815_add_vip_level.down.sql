-- https://stackoverflow.com/a/10404041

BEGIN;

-- Upgrade all VIP restricted items to moderators before removing them.
UPDATE channels SET mode = 'moderator' WHERE mode = 'vip';
UPDATE channels SET roll_level = 'moderator' WHERE roll_level = 'vip';
UPDATE channels SET filter_exempt_level = 'moderator' WHERE filter_exempt_level = 'vip';
UPDATE command_infos SET access_level = 'moderator' WHERE access_level = 'vip';

ALTER TYPE access_level RENAME TO access_level__;

CREATE TYPE access_level AS ENUM (
    'everyone',
    'subscriber',
    'moderator',
    'broadcaster',
    'admin'
);

ALTER TABLE channels
    ALTER COLUMN mode TYPE access_level USING mode::text::access_level,
    ALTER COLUMN roll_level TYPE access_level USING roll_level::text::access_level,
    ALTER COLUMN filter_exempt_level TYPE access_level USING filter_exempt_level::text::access_level,
    ALTER COLUMN filter_exempt_level SET DEFAULT 'subscriber'; -- Not strictly needed, but brings it back to the previous migration.

ALTER TABLE command_infos
    ALTER COLUMN access_level TYPE access_level USING access_level::text::access_level;

DROP TYPE access_level__;

COMMIT;
