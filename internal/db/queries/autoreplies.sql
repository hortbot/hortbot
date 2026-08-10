-- name: GetMaxAutoreplyNumber :one
SELECT COALESCE(max(num), 0)::integer
FROM autoreplies
WHERE channel_id = sqlc.arg(channel_id);

-- name: InsertAutoreply :exec
INSERT INTO autoreplies (
    channel_id,
    num,
    trigger,
    orig_pattern,
    response,
    count,
    creator,
    editor
)
VALUES (
    sqlc.arg(channel_id),
    sqlc.arg(num),
    sqlc.arg(trigger),
    sqlc.narg(orig_pattern),
    sqlc.arg(response),
    0,
    sqlc.arg(creator),
    sqlc.arg(editor)
);

-- name: GetAutoreplyForUpdate :one
SELECT *
FROM autoreplies
WHERE channel_id = sqlc.arg(channel_id)
  AND num = sqlc.arg(num)
FOR UPDATE;

-- name: DeleteAutoreply :exec
DELETE FROM autoreplies WHERE id = sqlc.arg(id);

-- name: UpdateAutoreplyResponse :exec
UPDATE autoreplies
SET response = sqlc.arg(response),
    editor = sqlc.arg(editor),
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);

-- name: UpdateAutoreplyPattern :exec
UPDATE autoreplies
SET trigger = sqlc.arg(trigger),
    orig_pattern = sqlc.narg(orig_pattern),
    editor = sqlc.arg(editor),
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);

-- name: ListAutoreplies :many
SELECT *
FROM autoreplies
WHERE channel_id = sqlc.arg(channel_id)
ORDER BY num;

-- name: ListAutoreplyMatchers :many
SELECT id, trigger, response, count
FROM autoreplies
WHERE channel_id = sqlc.arg(channel_id)
ORDER BY num;

-- name: UpdateAutoreplyCount :exec
UPDATE autoreplies SET count = sqlc.arg(count) WHERE id = sqlc.arg(id);

-- name: CompactAutoreplies :execrows
UPDATE autoreplies a
SET num = compacted.new_num
FROM (
    SELECT a2.id,
           a2.num,
           (ROW_NUMBER() OVER (ORDER BY a2.num)) + sqlc.arg(start_num)::integer - 1 AS new_num
    FROM autoreplies a2
    WHERE a2.channel_id = sqlc.arg(channel_id)
      AND a2.num >= sqlc.arg(start_num)
) compacted
WHERE compacted.id = a.id
  AND compacted.id != compacted.new_num;
