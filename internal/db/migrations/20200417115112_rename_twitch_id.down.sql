-- BREAKING CHANGE: Undoes the rename from user_id to twitch_id.
BEGIN;

ALTER TABLE channels RENAME COLUMN twitch_id TO user_id;
ALTER INDEX channels_twitch_id_idx RENAME TO channels_user_id_idx;

COMMIT;
