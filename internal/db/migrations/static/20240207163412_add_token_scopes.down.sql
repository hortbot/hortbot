BEGIN;

ALTER TABLE twitch_tokens DROP COLUMN scopes;

COMMIT;
