-- name: BotStateCurrentTime :one
SELECT clock_timestamp()::timestamptz AS current_time;

-- name: BotStateAcquireAdvisoryLock :exec
SELECT pg_advisory_xact_lock(sqlc.arg(class_id), sqlc.arg(object_id));

-- name: BotStateSetAuthState :exec
INSERT INTO web_auth_states (key, value, expires_at)
VALUES (sqlc.arg(key), sqlc.arg(value), sqlc.arg(expires_at))
ON CONFLICT (key) DO UPDATE
SET value = excluded.value,
    expires_at = excluded.expires_at;

-- name: BotStateTakeAuthState :one
DELETE FROM web_auth_states
WHERE key = sqlc.arg(key) AND expires_at > sqlc.arg(now)
RETURNING value;

-- name: BotStateMarkCommandCooldown :exec
INSERT INTO bot_command_cooldowns (channel, command_key, expires_at)
VALUES (sqlc.arg(channel), sqlc.arg(command_key), sqlc.arg(expires_at))
ON CONFLICT (channel, command_key) DO UPDATE
SET expires_at = excluded.expires_at;

-- name: BotStateCheckAndMarkCommandCooldown :one
INSERT INTO bot_command_cooldowns (channel, command_key, expires_at)
VALUES (sqlc.arg(channel), sqlc.arg(command_key), sqlc.arg(expires_at))
ON CONFLICT (channel, command_key) DO UPDATE
SET expires_at = excluded.expires_at
WHERE bot_command_cooldowns.expires_at <= sqlc.arg(now)
RETURNING true;

-- name: BotStateCheckAndMarkRepeatCooldown :one
INSERT INTO bot_repeat_cooldowns (channel, repeated_command_id, expires_at)
VALUES (sqlc.arg(channel), sqlc.arg(repeated_command_id), sqlc.arg(expires_at))
ON CONFLICT (channel, repeated_command_id) DO UPDATE
SET expires_at = excluded.expires_at
WHERE bot_repeat_cooldowns.expires_at <= sqlc.arg(now)
RETURNING true;

-- name: BotStateCheckAndMarkScheduledCooldown :one
INSERT INTO bot_scheduled_command_cooldowns (channel, scheduled_command_id, expires_at)
VALUES (sqlc.arg(channel), sqlc.arg(scheduled_command_id), sqlc.arg(expires_at))
ON CONFLICT (channel, scheduled_command_id) DO UPDATE
SET expires_at = excluded.expires_at
WHERE bot_scheduled_command_cooldowns.expires_at <= sqlc.arg(now)
RETURNING true;

-- name: BotStateCheckAndMarkAutoreplyCooldown :one
INSERT INTO bot_autoreply_cooldowns (channel, autoreply_id, expires_at)
VALUES (sqlc.arg(channel), sqlc.arg(autoreply_id), sqlc.arg(expires_at))
ON CONFLICT (channel, autoreply_id) DO UPDATE
SET expires_at = excluded.expires_at
WHERE bot_autoreply_cooldowns.expires_at <= sqlc.arg(now)
RETURNING true;

-- name: BotStateGrantLinkPermit :exec
INSERT INTO bot_link_permits (channel, user_id, expires_at)
VALUES (sqlc.arg(channel), sqlc.arg(user_id), sqlc.arg(expires_at))
ON CONFLICT (channel, user_id) DO UPDATE
SET expires_at = excluded.expires_at;

-- name: BotStateConsumeLinkPermit :execrows
DELETE FROM bot_link_permits
WHERE channel = sqlc.arg(channel)
  AND user_id = sqlc.arg(user_id)
  AND expires_at > sqlc.arg(now);

-- name: BotStateGetConfirmationExpiry :one
SELECT expires_at
FROM bot_confirmations
WHERE channel = sqlc.arg(channel)
  AND user_id = sqlc.arg(user_id)
  AND confirmation_key = sqlc.arg(confirmation_key);

-- name: BotStateDeleteConfirmation :exec
DELETE FROM bot_confirmations
WHERE channel = sqlc.arg(channel)
  AND user_id = sqlc.arg(user_id)
  AND confirmation_key = sqlc.arg(confirmation_key);

-- name: BotStateUpsertConfirmation :exec
INSERT INTO bot_confirmations (channel, user_id, confirmation_key, expires_at)
VALUES (
    sqlc.arg(channel),
    sqlc.arg(user_id),
    sqlc.arg(confirmation_key),
    sqlc.arg(expires_at)
)
ON CONFLICT (channel, user_id, confirmation_key) DO UPDATE
SET expires_at = excluded.expires_at;

-- name: BotStateGetFilterWarningExpiry :one
SELECT expires_at
FROM bot_filter_warnings
WHERE channel = sqlc.arg(channel)
  AND user_id = sqlc.arg(user_id)
  AND filter_name = sqlc.arg(filter_name);

-- name: BotStateUpsertFilterWarning :exec
INSERT INTO bot_filter_warnings (channel, user_id, filter_name, expires_at)
VALUES (
    sqlc.arg(channel),
    sqlc.arg(user_id),
    sqlc.arg(filter_name),
    sqlc.arg(expires_at)
)
ON CONFLICT (channel, user_id, filter_name) DO UPDATE
SET expires_at = excluded.expires_at;

-- name: BotStateCleanupCommandCooldowns :exec
DELETE FROM bot_command_cooldowns WHERE expires_at < now();

-- name: BotStateCleanupRepeatCooldowns :exec
DELETE FROM bot_repeat_cooldowns WHERE expires_at < now();

-- name: BotStateCleanupScheduledCooldowns :exec
DELETE FROM bot_scheduled_command_cooldowns WHERE expires_at < now();

-- name: BotStateCleanupAutoreplyCooldowns :exec
DELETE FROM bot_autoreply_cooldowns WHERE expires_at < now();

-- name: BotStateCleanupLinkPermits :exec
DELETE FROM bot_link_permits WHERE expires_at < now();

-- name: BotStateCleanupConfirmations :exec
DELETE FROM bot_confirmations WHERE expires_at < now();

-- name: BotStateCleanupFilterWarnings :exec
DELETE FROM bot_filter_warnings WHERE expires_at < now();

-- name: BotStateCleanupAuthStates :exec
DELETE FROM web_auth_states WHERE expires_at < now();

-- name: BotStateDumpCommandCooldowns :many
SELECT (channel || '/' || command_key)::text AS key, ''::text AS value, expires_at
FROM bot_command_cooldowns
ORDER BY channel, command_key;

-- name: BotStateDumpRepeatCooldowns :many
SELECT (channel || '/' || repeated_command_id)::text AS key, ''::text AS value, expires_at
FROM bot_repeat_cooldowns
ORDER BY channel, repeated_command_id;

-- name: BotStateDumpScheduledCooldowns :many
SELECT (channel || '/' || scheduled_command_id)::text AS key, ''::text AS value, expires_at
FROM bot_scheduled_command_cooldowns
ORDER BY channel, scheduled_command_id;

-- name: BotStateDumpAutoreplyCooldowns :many
SELECT (channel || '/' || autoreply_id)::text AS key, ''::text AS value, expires_at
FROM bot_autoreply_cooldowns
ORDER BY channel, autoreply_id;

-- name: BotStateDumpLinkPermits :many
SELECT (channel || '/' || user_id)::text AS key, ''::text AS value, expires_at
FROM bot_link_permits
ORDER BY channel, user_id;

-- name: BotStateDumpConfirmations :many
SELECT (channel || '/' || user_id || '/' || confirmation_key)::text AS key,
       ''::text AS value,
       expires_at
FROM bot_confirmations
ORDER BY channel, user_id, confirmation_key;

-- name: BotStateDumpFilterWarnings :many
SELECT (channel || '/' || user_id || '/' || filter_name)::text AS key,
       ''::text AS value,
       expires_at
FROM bot_filter_warnings
ORDER BY channel, user_id, filter_name;

-- name: BotStateDumpAuthStates :many
SELECT key, encode(value, 'escape') AS value, expires_at
FROM web_auth_states
ORDER BY key;

-- name: BotStateRaffleAdd :exec
INSERT INTO bot_raffle_entries (channel, user_id)
VALUES (sqlc.arg(channel), sqlc.arg(user_id))
ON CONFLICT (channel, user_id) DO NOTHING;

-- name: BotStateRaffleReset :exec
DELETE FROM bot_raffle_entries WHERE channel = sqlc.arg(channel);

-- name: BotStateRaffleCount :one
SELECT count(*) FROM bot_raffle_entries WHERE channel = sqlc.arg(channel);

-- name: BotStateListRaffleEntriesForUpdate :many
SELECT user_id
FROM bot_raffle_entries
WHERE channel = sqlc.arg(channel)
ORDER BY user_id
FOR UPDATE;

-- name: BotStateDeleteRaffleEntries :exec
DELETE FROM bot_raffle_entries
WHERE channel = sqlc.arg(channel)
  AND user_id = ANY(sqlc.arg(user_ids)::text[]);

-- name: BotStateAddBuiltinUsageStat :exec
INSERT INTO bot_builtin_usage_stats (name, count)
VALUES (sqlc.arg(name), sqlc.arg(count))
ON CONFLICT (name) DO UPDATE
SET count = bot_builtin_usage_stats.count + excluded.count;

-- name: BotStateMergeBuiltinUsageStat :exec
INSERT INTO bot_builtin_usage_stats (name, count)
VALUES (sqlc.arg(name), sqlc.arg(count))
ON CONFLICT (name) DO UPDATE
SET count = GREATEST(bot_builtin_usage_stats.count, excluded.count);

-- name: BotStateListBuiltinUsageStats :many
SELECT name, count FROM bot_builtin_usage_stats;

-- name: BotStateAddActionUsageStat :exec
INSERT INTO bot_action_usage_stats (name, count)
VALUES (sqlc.arg(name), sqlc.arg(count))
ON CONFLICT (name) DO UPDATE
SET count = bot_action_usage_stats.count + excluded.count;

-- name: BotStateMergeActionUsageStat :exec
INSERT INTO bot_action_usage_stats (name, count)
VALUES (sqlc.arg(name), sqlc.arg(count))
ON CONFLICT (name) DO UPDATE
SET count = GREATEST(bot_action_usage_stats.count, excluded.count);

-- name: BotStateListActionUsageStats :many
SELECT name, count FROM bot_action_usage_stats;
