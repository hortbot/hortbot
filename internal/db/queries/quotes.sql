-- name: GetMaxQuoteNumber :one
SELECT COALESCE(max(num), 0)::integer FROM quotes WHERE channel_id = sqlc.arg(channel_id);

-- name: InsertQuote :exec
INSERT INTO quotes (channel_id, num, quote, creator, editor)
VALUES (
    sqlc.arg(channel_id),
    sqlc.arg(num),
    sqlc.arg(quote),
    sqlc.arg(creator),
    sqlc.arg(editor)
);

-- name: GetQuoteByNumber :one
SELECT * FROM quotes
WHERE channel_id = sqlc.arg(channel_id) AND num = sqlc.arg(num);

-- name: GetQuoteByNumberForUpdate :one
SELECT * FROM quotes
WHERE channel_id = sqlc.arg(channel_id) AND num = sqlc.arg(num)
FOR UPDATE;

-- name: GetQuoteByText :one
SELECT * FROM quotes
WHERE channel_id = sqlc.arg(channel_id) AND quote = sqlc.arg(quote);

-- name: QuoteExistsAfterNumber :one
SELECT EXISTS (
    SELECT 1 FROM quotes
    WHERE channel_id = sqlc.arg(channel_id) AND num > sqlc.arg(num)
);

-- name: DeleteQuote :exec
DELETE FROM quotes WHERE id = sqlc.arg(id);

-- name: UpdateQuote :exec
UPDATE quotes
SET quote = sqlc.arg(quote),
    editor = sqlc.arg(editor),
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);

-- name: GetRandomQuote :one
SELECT * FROM quotes
WHERE channel_id = sqlc.arg(channel_id)
ORDER BY random()
LIMIT 1;

-- name: SearchQuoteNumbers :many
SELECT num FROM quotes
WHERE channel_id = sqlc.arg(channel_id)
  AND quote ILIKE sqlc.arg(pattern)
ORDER BY num;

-- name: ListQuotes :many
SELECT * FROM quotes
WHERE channel_id = sqlc.arg(channel_id)
ORDER BY num;

-- name: CompactQuotes :execrows
UPDATE quotes q
SET num = compacted.new_num
FROM (
    SELECT q2.id,
           q2.num,
           (ROW_NUMBER() OVER (ORDER BY q2.num)) + sqlc.arg(start_num)::integer - 1 AS new_num
    FROM quotes q2
    WHERE q2.channel_id = sqlc.arg(channel_id)
      AND q2.num >= sqlc.arg(start_num)
) compacted
WHERE compacted.id = q.id
  AND compacted.num != compacted.new_num;
