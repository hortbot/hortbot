BEGIN;

ALTER TABLE twitch_tokens ADD COLUMN scopes text[];

COMMIT;
