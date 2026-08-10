-- name: GetVariable :one
SELECT *
FROM variables
WHERE channel_id = sqlc.arg(channel_id)
  AND name = sqlc.arg(name);

-- name: GetVariableByChannelName :one
SELECT variables.*
FROM variables
JOIN channels ON channels.id = variables.channel_id
WHERE channels.name = sqlc.arg(channel_name)
  AND variables.name = sqlc.arg(name);

-- name: ListVariables :many
SELECT *
FROM variables
WHERE channel_id = sqlc.arg(channel_id)
ORDER BY name;

-- name: UpsertVariable :exec
INSERT INTO variables (channel_id, name, value)
VALUES (sqlc.arg(channel_id), sqlc.arg(name), sqlc.arg(value))
ON CONFLICT (channel_id, name) DO UPDATE
SET value = excluded.value,
    updated_at = statement_timestamp();

-- name: UpdateVariableValue :exec
UPDATE variables
SET value = sqlc.arg(value),
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);

-- name: DeleteVariable :exec
DELETE FROM variables
WHERE channel_id = sqlc.arg(channel_id)
  AND name = sqlc.arg(name);
