-- name: GetTwitchTokenByID :one
SELECT * FROM twitch_tokens WHERE twitch_id = sqlc.arg(twitch_id);

-- name: GetTwitchTokenByBotName :one
SELECT * FROM twitch_tokens WHERE bot_name = sqlc.arg(bot_name)::text;

-- name: ListTwitchTokens :many
SELECT * FROM twitch_tokens ORDER BY twitch_id;

-- name: ListBotTwitchTokens :many
SELECT * FROM twitch_tokens WHERE bot_name IS NOT NULL ORDER BY bot_name;

-- name: ListModerationBotTwitchTokens :many
SELECT *
FROM twitch_tokens
WHERE bot_name IS NOT NULL
  AND scopes @> ARRAY['user:read:moderated_channels']
ORDER BY bot_name;

-- name: UpsertTwitchToken :one
INSERT INTO twitch_tokens (
    twitch_id,
    bot_name,
    access_token,
    token_type,
    refresh_token,
    expiry,
    scopes
)
VALUES (
    sqlc.arg(twitch_id),
    sqlc.narg(bot_name),
    sqlc.arg(access_token),
    sqlc.arg(token_type),
    sqlc.arg(refresh_token),
    sqlc.arg(expiry),
    sqlc.arg(scopes)::text[]
)
ON CONFLICT (twitch_id) DO UPDATE
SET bot_name = excluded.bot_name,
    access_token = excluded.access_token,
    token_type = excluded.token_type,
    refresh_token = excluded.refresh_token,
    expiry = excluded.expiry,
    scopes = excluded.scopes,
    updated_at = statement_timestamp()
RETURNING *;

-- name: UpsertTwitchTokenPreservingMetadata :one
INSERT INTO twitch_tokens (
    twitch_id,
    access_token,
    token_type,
    refresh_token,
    expiry
)
VALUES (
    sqlc.arg(twitch_id),
    sqlc.arg(access_token),
    sqlc.arg(token_type),
    sqlc.arg(refresh_token),
    sqlc.arg(expiry)
)
ON CONFLICT (twitch_id) DO UPDATE
SET access_token = excluded.access_token,
    token_type = excluded.token_type,
    refresh_token = excluded.refresh_token,
    expiry = excluded.expiry,
    updated_at = statement_timestamp()
RETURNING *;

-- name: DeleteTwitchTokenByID :exec
DELETE FROM twitch_tokens WHERE twitch_id = sqlc.arg(twitch_id);

-- name: IsBlockedUser :one
SELECT EXISTS (
    SELECT 1 FROM blocked_users WHERE twitch_id = sqlc.arg(twitch_id)
);

-- name: UpsertBlockedUser :exec
INSERT INTO blocked_users (twitch_id)
VALUES (sqlc.arg(twitch_id))
ON CONFLICT (twitch_id) DO NOTHING;

-- name: DeleteBlockedUser :exec
DELETE FROM blocked_users WHERE twitch_id = sqlc.arg(twitch_id);

-- name: IsModeratedChannel :one
SELECT EXISTS (
    SELECT 1
    FROM moderated_channels
    WHERE broadcaster_id = sqlc.arg(broadcaster_id)
      AND bot_name = sqlc.arg(bot_name)
);

-- name: LockModeratedChannels :exec
LOCK TABLE moderated_channels IN EXCLUSIVE MODE;

-- name: UpsertModeratedChannel :exec
INSERT INTO moderated_channels (
    bot_name,
    broadcaster_id,
    broadcaster_login,
    broadcaster_name,
    updated_at
)
VALUES (
    sqlc.arg(bot_name),
    sqlc.arg(broadcaster_id),
    sqlc.arg(broadcaster_login),
    sqlc.arg(broadcaster_name),
    sqlc.arg(updated_at)
)
ON CONFLICT (bot_name, broadcaster_id) DO UPDATE
SET broadcaster_login = excluded.broadcaster_login,
    broadcaster_name = excluded.broadcaster_name,
    updated_at = excluded.updated_at;

-- name: DeleteStaleModeratedChannels :exec
DELETE FROM moderated_channels
WHERE bot_name = sqlc.arg(bot_name)
  AND updated_at < sqlc.arg(updated_before);
