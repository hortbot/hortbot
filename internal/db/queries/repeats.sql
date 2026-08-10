-- name: GetRepeatedCommandByInfo :one
SELECT * FROM repeated_commands WHERE command_info_id = sqlc.arg(command_info_id);

-- name: InsertRepeatedCommand :one
INSERT INTO repeated_commands (
  created_at, updated_at, channel_id, command_info_id, enabled, delay,
  message_diff, last_count, creator, editor
)
VALUES (
  sqlc.arg(now), sqlc.arg(now), sqlc.arg(channel_id), sqlc.arg(command_info_id),
  true, sqlc.arg(delay),
  sqlc.arg(message_diff), sqlc.arg(last_count), sqlc.arg(creator), sqlc.arg(editor)
)
RETURNING *;

-- name: UpdateRepeatedCommand :one
UPDATE repeated_commands
SET enabled = sqlc.arg(enabled),
    delay = sqlc.arg(delay),
    message_diff = sqlc.arg(message_diff),
    last_count = sqlc.arg(last_count),
    init_timestamp = NULL,
    editor = sqlc.arg(editor),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteRepeatedCommand :exec
DELETE FROM repeated_commands WHERE id = sqlc.arg(id);

-- name: ListRepeatedCommands :many
SELECT r.*
FROM repeated_commands r
WHERE r.channel_id = sqlc.arg(channel_id);

-- name: ListRepeatedCommandsWithNames :many
SELECT r.*, ci.name
FROM repeated_commands r
JOIN command_infos ci ON ci.id = r.command_info_id
WHERE r.channel_id = sqlc.arg(channel_id)
ORDER BY ci.name;

-- name: ListRepeatedCommandsForWeb :many
SELECT r.enabled, r.delay, r.message_diff, ci.name
FROM repeated_commands r
JOIN command_infos ci ON ci.id = r.command_info_id
WHERE r.channel_id = sqlc.arg(channel_id)
ORDER BY r.enabled, ci.name;

-- name: ListScheduledCommandsForWeb :many
SELECT s.enabled, s.cron_expression, s.message_diff, ci.name
FROM scheduled_commands s
JOIN command_infos ci ON ci.id = s.command_info_id
WHERE s.channel_id = sqlc.arg(channel_id)
ORDER BY s.enabled, ci.name;

-- name: GetRepeatedCommandStatus :one
SELECT r.enabled,
       c.active,
       c.message_count >= (r.last_count + r.message_diff) AS ready
FROM repeated_commands r
JOIN channels c ON c.id = r.channel_id
WHERE r.id = sqlc.arg(id);

-- name: GetScheduledCommandStatus :one
SELECT s.enabled,
       c.active,
       c.message_count >= (s.last_count + s.message_diff) AS ready
FROM scheduled_commands s
JOIN channels c ON c.id = s.channel_id
WHERE s.id = sqlc.arg(id);

-- name: GetRepeatedCommandForRun :one
SELECT * FROM repeated_commands
WHERE id = sqlc.arg(id) AND enabled
FOR UPDATE;

-- name: GetScheduledCommandForRun :one
SELECT * FROM scheduled_commands
WHERE id = sqlc.arg(id) AND enabled
FOR UPDATE;

-- name: GetCommandInfoByIDForUpdate :one
SELECT * FROM command_infos WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: UpdateRepeatedCommandLastCount :exec
UPDATE repeated_commands SET last_count = sqlc.arg(last_count) WHERE id = sqlc.arg(id);

-- name: UpdateScheduledCommandLastCount :exec
UPDATE scheduled_commands SET last_count = sqlc.arg(last_count) WHERE id = sqlc.arg(id);

-- name: ListActiveRepeatedCommands :many
SELECT r.*
FROM repeated_commands r
JOIN channels c ON r.channel_id = c.id
LEFT JOIN twitch_tokens tt ON tt.twitch_id = c.twitch_id
LEFT JOIN moderated_channels m ON m.broadcaster_id = c.twitch_id AND m.bot_name = c.bot_name
WHERE r.enabled AND c.active AND ('channel:bot' = ANY(tt.scopes) OR m.id IS NOT NULL);

-- name: ListActiveScheduledCommands :many
SELECT s.*
FROM scheduled_commands s
JOIN channels c ON s.channel_id = c.id
LEFT JOIN twitch_tokens tt ON tt.twitch_id = c.twitch_id
LEFT JOIN moderated_channels m ON m.broadcaster_id = c.twitch_id AND m.bot_name = c.bot_name
WHERE s.enabled AND c.active AND ('channel:bot' = ANY(tt.scopes) OR m.id IS NOT NULL);

-- name: GetScheduledCommandByInfo :one
SELECT * FROM scheduled_commands WHERE command_info_id = sqlc.arg(command_info_id);

-- name: InsertScheduledCommand :one
INSERT INTO scheduled_commands (
  created_at, updated_at, channel_id, command_info_id, enabled,
  cron_expression, message_diff, last_count, creator, editor
)
VALUES (
  sqlc.arg(now), sqlc.arg(now), sqlc.arg(channel_id), sqlc.arg(command_info_id),
  true, sqlc.arg(cron_expression), sqlc.arg(message_diff),
  sqlc.arg(last_count), sqlc.arg(creator), sqlc.arg(editor)
)
RETURNING *;

-- name: UpdateScheduledCommand :one
UPDATE scheduled_commands
SET enabled = sqlc.arg(enabled),
    cron_expression = sqlc.arg(cron_expression),
    message_diff = sqlc.arg(message_diff),
    last_count = sqlc.arg(last_count),
    editor = sqlc.arg(editor),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteScheduledCommand :exec
DELETE FROM scheduled_commands WHERE id = sqlc.arg(id);

-- name: ListScheduledCommands :many
SELECT s.*
FROM scheduled_commands s
WHERE s.channel_id = sqlc.arg(channel_id);

-- name: ListScheduledCommandsWithNames :many
SELECT s.*, ci.name
FROM scheduled_commands s
JOIN command_infos ci ON ci.id = s.command_info_id
WHERE s.channel_id = sqlc.arg(channel_id)
ORDER BY ci.name;
