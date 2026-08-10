-- name: InsertHighlight :exec
INSERT INTO highlights (
    channel_id,
    highlighted_at,
    started_at,
    status,
    game
)
VALUES (
    sqlc.arg(channel_id),
    sqlc.arg(highlighted_at),
    sqlc.narg(started_at),
    sqlc.arg(status),
    sqlc.arg(game)
);

-- name: DeleteHighlightsByChannel :exec
DELETE FROM highlights WHERE channel_id = sqlc.arg(channel_id);

-- name: ListHighlights :many
SELECT *
FROM highlights
WHERE channel_id = sqlc.arg(channel_id)
ORDER BY created_at;

-- name: ListRecentHighlights :many
SELECT *
FROM highlights
WHERE channel_id = sqlc.arg(channel_id)
  AND highlighted_at > sqlc.arg(highlighted_after)
ORDER BY highlighted_at;
