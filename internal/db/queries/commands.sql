-- name: FindCommand :one
SELECT ci.*, cc.message
FROM command_infos ci
LEFT JOIN custom_commands cc ON cc.id = ci.custom_command_id
WHERE ci.channel_id = sqlc.arg(channel_id)
  AND ci.name = sqlc.arg(name);

-- name: FindCommandForUpdate :one
SELECT ci.*, cc.message
FROM command_infos ci
LEFT JOIN custom_commands cc ON cc.id = ci.custom_command_id
WHERE ci.channel_id = sqlc.arg(channel_id)
  AND ci.name = sqlc.arg(name)
FOR UPDATE OF ci;

-- name: GetCommandInfo :one
SELECT *
FROM command_infos
WHERE channel_id = sqlc.arg(channel_id)
  AND name = sqlc.arg(name);

-- name: ListCommandInfos :many
SELECT * FROM command_infos
WHERE channel_id = sqlc.arg(channel_id)
ORDER BY name;

-- name: GetCommandInfoForUpdate :one
SELECT *
FROM command_infos
WHERE channel_id = sqlc.arg(channel_id)
  AND name = sqlc.arg(name)
FOR UPDATE;

-- name: CommandInfoExists :one
SELECT EXISTS(
  SELECT 1 FROM command_infos
  WHERE channel_id = sqlc.arg(channel_id)
    AND name = sqlc.arg(name)
);

-- name: GetCustomCommand :one
SELECT * FROM custom_commands WHERE id = sqlc.arg(id);

-- name: GetCustomCommandForUpdate :one
SELECT * FROM custom_commands WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: ListCustomCommands :many
SELECT * FROM custom_commands WHERE channel_id = sqlc.arg(channel_id);

-- name: InsertCustomCommand :one
INSERT INTO custom_commands (channel_id, message)
VALUES (sqlc.arg(channel_id), sqlc.arg(message))
RETURNING *;

-- name: UpdateCustomCommandMessage :exec
UPDATE custom_commands
SET message = sqlc.arg(message),
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);

-- name: ListCustomCommandsForWeb :many
SELECT cc.message, cc.updated_at, ci.name, ci.access_level, ci.count, ci.editor
FROM custom_commands cc
JOIN command_infos ci ON ci.custom_command_id = cc.id
WHERE cc.channel_id = sqlc.arg(channel_id)
ORDER BY ci.name;

-- name: ListCommandListsForWeb :many
SELECT cl.items, cl.updated_at, ci.name, ci.access_level, ci.count, ci.editor
FROM command_lists cl
JOIN command_infos ci ON ci.command_list_id = cl.id
WHERE cl.channel_id = sqlc.arg(channel_id)
ORDER BY ci.name;

-- name: UpdateCommandInfoCount :exec
UPDATE command_infos SET count = sqlc.arg(count) WHERE id = sqlc.arg(id);

-- name: UpdateCommandInfoUsage :exec
UPDATE command_infos
SET count = sqlc.arg(count),
    last_used = sqlc.arg(last_used)
WHERE id = sqlc.arg(id);

-- name: GetRepeatedCommandIDByInfo :one
SELECT id FROM repeated_commands WHERE command_info_id = sqlc.arg(command_info_id);

-- name: GetScheduledCommandIDByInfo :one
SELECT id FROM scheduled_commands WHERE command_info_id = sqlc.arg(command_info_id);

-- name: DeleteRepeatedCommandByInfo :exec
DELETE FROM repeated_commands WHERE command_info_id = sqlc.arg(command_info_id);

-- name: DeleteScheduledCommandByInfo :exec
DELETE FROM scheduled_commands WHERE command_info_id = sqlc.arg(command_info_id);

-- name: DeleteCommandInfo :exec
DELETE FROM command_infos WHERE id = sqlc.arg(id);

-- name: DeleteCustomCommand :exec
DELETE FROM custom_commands WHERE id = sqlc.arg(id);

-- name: DeleteCommandList :exec
DELETE FROM command_lists WHERE id = sqlc.arg(id);

-- name: InsertCommandList :one
INSERT INTO command_lists (channel_id, items)
VALUES (sqlc.arg(channel_id), ARRAY[]::text[])
RETURNING *;

-- name: GetCommandList :one
SELECT * FROM command_lists WHERE id = sqlc.arg(id);

-- name: GetCommandListForUpdate :one
SELECT * FROM command_lists WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: ListCommandLists :many
SELECT * FROM command_lists WHERE channel_id = sqlc.arg(channel_id);

-- name: UpdateCommandListItems :exec
UPDATE command_lists
SET items = sqlc.arg(items)::text[],
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);

-- name: UpdateCommandInfoEditor :exec
UPDATE command_infos
SET editor = sqlc.arg(editor),
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);

-- name: InsertCommandInfo :one
INSERT INTO command_infos (
  channel_id, name, access_level, count, creator, editor,
  custom_command_id, command_list_id
)
VALUES (
  sqlc.arg(channel_id), sqlc.arg(name), sqlc.arg(access_level), 0,
  sqlc.arg(creator), sqlc.arg(editor),
  sqlc.narg(custom_command_id)::bigint, sqlc.narg(command_list_id)::bigint
)
RETURNING *;

-- name: UpdateCommandInfoAccess :exec
UPDATE command_infos
SET access_level = sqlc.arg(access_level),
    editor = sqlc.arg(editor),
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);

-- name: RenameCommandInfo :exec
UPDATE command_infos
SET name = sqlc.arg(name),
    editor = sqlc.arg(editor),
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);
