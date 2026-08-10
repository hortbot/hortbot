-- name: AcquireTwitchAdvisoryLock :exec
SELECT pg_advisory_xact_lock(sqlc.arg(twitch_id));

-- name: GetChannelIDByName :one
SELECT id
FROM channels
WHERE name = sqlc.arg(name);

-- name: CountChannels :one
SELECT COUNT(*) FROM channels;

-- name: GetChannelByID :one
SELECT * FROM channels WHERE id = sqlc.arg(id);

-- name: GetChannelByTwitchIDForUpdate :one
SELECT * FROM channels WHERE twitch_id = sqlc.arg(twitch_id) FOR UPDATE;

-- name: GetChannelByNameForUpdate :one
SELECT * FROM channels WHERE name = sqlc.arg(name) FOR UPDATE;

-- name: GetChannelByName :one
SELECT * FROM channels WHERE name = sqlc.arg(name);

-- name: InsertDefaultChannel :one
INSERT INTO channels (
  twitch_id, name, display_name, bot_name, active, prefix, mode,
  message_count, last_fm, parse_youtube, extra_life_id, raffle_enabled,
  steam_id, urban_enabled,
  should_moderate, enable_warnings, subs_may_link, timeout_duration,
  display_warnings, enable_filters, filter_links, filter_caps,
  filter_caps_min_chars, filter_emotes, filter_emotes_single, filter_symbols,
  filter_me, filter_banned_phrases, sub_message, sub_message_enabled,
  resub_message, resub_message_enabled,
  roll_level, roll_cooldown, roll_default, filter_caps_percentage,
  filter_caps_min_caps, filter_symbols_percentage, filter_symbols_min_symbols,
  filter_max_length, filter_emotes_max, tweet, filter_exempt_level
)
VALUES (
  sqlc.arg(twitch_id), sqlc.arg(name), sqlc.arg(display_name), sqlc.arg(bot_name),
  true, '!', 'everyone', 0, '', false, 0, false, '', false,
  true, true, true, 600, false, false, false, false, 0, false, false, false,
  false, false, '', false, '', false, 'subscriber', 10, 20,
  50, 6, 50, 5, 500, 4,
  'Check out (_CHANNEL_URL_) playing (_GAME_) on @Twitch!', 'subscriber'
)
RETURNING *;

-- name: UpdateChannelMembership :exec
UPDATE channels
SET active = sqlc.arg(active),
    bot_name = sqlc.arg(bot_name),
    name = sqlc.arg(name),
    display_name = sqlc.arg(display_name),
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);

-- name: UpdateChannelActive :exec
UPDATE channels SET active = sqlc.arg(active), updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);

-- name: GetActiveChannelByName :one
SELECT c.*
FROM channels c
LEFT JOIN twitch_tokens tt ON tt.twitch_id = c.twitch_id
LEFT JOIN moderated_channels m ON m.broadcaster_id = c.twitch_id AND m.bot_name = c.bot_name
WHERE c.active
  AND ('channel:bot' = ANY(tt.scopes) OR m.id IS NOT NULL)
  AND c.name = sqlc.arg(name);

-- name: GetChannelBotByName :one
SELECT id, bot_name
FROM channels
WHERE name = sqlc.arg(name);

-- name: UpdateChannelBotName :exec
UPDATE channels
SET bot_name = sqlc.arg(bot_name),
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);

-- name: UpdateChannelIdentity :exec
UPDATE channels
SET name = sqlc.arg(name),
    display_name = sqlc.arg(display_name),
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);

-- name: DeleteScheduledCommandsByChannel :exec
DELETE FROM scheduled_commands WHERE channel_id = sqlc.arg(channel_id);

-- name: DeleteRepeatedCommandsByChannel :exec
DELETE FROM repeated_commands WHERE channel_id = sqlc.arg(channel_id);

-- name: DeleteCommandInfosByChannel :exec
DELETE FROM command_infos WHERE channel_id = sqlc.arg(channel_id);

-- name: DeleteCommandListsByChannel :exec
DELETE FROM command_lists WHERE channel_id = sqlc.arg(channel_id);

-- name: DeleteVariablesByChannel :exec
DELETE FROM variables WHERE channel_id = sqlc.arg(channel_id);

-- name: DeleteAutorepliesByChannel :exec
DELETE FROM autoreplies WHERE channel_id = sqlc.arg(channel_id);

-- name: DeleteQuotesByChannel :exec
DELETE FROM quotes WHERE channel_id = sqlc.arg(channel_id);

-- name: DeleteCustomCommandsByChannel :exec
DELETE FROM custom_commands WHERE channel_id = sqlc.arg(channel_id);

-- name: DeleteChannel :exec
DELETE FROM channels WHERE id = sqlc.arg(id);

-- name: UpdateChannelActivity :exec
UPDATE channels
SET message_count = sqlc.arg(message_count),
    last_seen = sqlc.arg(last_seen)
WHERE id = sqlc.arg(id);
