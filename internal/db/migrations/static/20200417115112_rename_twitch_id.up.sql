-- BREAKING CHANGE: Renames user_id to twitch_id for consistency.
BEGIN;

ALTER TABLE channels RENAME COLUMN user_id TO twitch_id;
ALTER INDEX channels_user_id_idx RENAME TO channels_twitch_id_idx;

COMMIT;
