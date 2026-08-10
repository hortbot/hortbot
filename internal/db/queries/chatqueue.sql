-- name: ChatQueueEnsureKey :exec
INSERT INTO chat_message_queue_keys (broadcaster_login)
VALUES (sqlc.arg(broadcaster_login))
ON CONFLICT (broadcaster_login) DO NOTHING;

-- name: ChatQueueEnqueue :execrows
INSERT INTO chat_message_queue (
    message_id,
    broadcaster_login,
    message_timestamp,
    enqueued_at,
    payload
)
VALUES (
    sqlc.arg(message_id),
    sqlc.arg(broadcaster_login),
    sqlc.arg(message_timestamp),
    sqlc.arg(enqueued_at),
    sqlc.arg(payload)
)
ON CONFLICT (message_id) DO NOTHING;

-- name: ChatQueueNotify :exec
SELECT pg_notify(sqlc.arg(channel), '');

-- name: ChatQueueClaim :one
SELECT
    q.message_id,
    q.broadcaster_login,
    q.message_timestamp,
    q.enqueued_at,
    q.payload
FROM chat_message_queue AS q
JOIN chat_message_queue_keys AS k USING (broadcaster_login)
WHERE q.completed_at IS NULL
  AND q.failed_at IS NULL
  AND (q.lease_until IS NULL OR q.lease_until <= NOW())
  AND (k.lease_until IS NULL OR k.lease_until <= NOW())
ORDER BY q.enqueued_at, q.message_id
FOR UPDATE OF q, k SKIP LOCKED
LIMIT 1;

-- name: ChatQueueLeaseKey :one
UPDATE chat_message_queue_keys
SET lease_token = sqlc.arg(lease_token)::text,
    lease_until = NOW() + (sqlc.arg(lease_microseconds)::bigint * INTERVAL '1 microsecond')
WHERE broadcaster_login = sqlc.arg(broadcaster_login)
RETURNING COALESCE(lease_until, NOW())::timestamptz AS lease_until;

-- name: ChatQueueLeaseMessage :exec
UPDATE chat_message_queue
SET lease_token = sqlc.arg(lease_token)::text,
    lease_until = sqlc.arg(lease_until)::timestamptz
WHERE message_id = sqlc.arg(message_id);

-- name: ChatQueueComplete :execrows
UPDATE chat_message_queue
SET completed_at = NOW(),
    lease_token = NULL,
    lease_until = NULL
WHERE message_id = sqlc.arg(message_id)
  AND lease_token = sqlc.arg(lease_token)::text
  AND completed_at IS NULL
  AND failed_at IS NULL;

-- name: ChatQueueReleaseKey :exec
UPDATE chat_message_queue_keys
SET lease_token = NULL,
    lease_until = NULL
WHERE broadcaster_login = sqlc.arg(broadcaster_login)
  AND lease_token = sqlc.arg(lease_token)::text;

-- name: ChatQueueFail :execrows
UPDATE chat_message_queue
SET failed_at = NOW(),
    last_error = sqlc.arg(last_error)::text,
    lease_token = NULL,
    lease_until = NULL
WHERE message_id = sqlc.arg(message_id)
  AND lease_token = sqlc.arg(lease_token)::text;

-- name: ChatQueueDeleteStale :execrows
WITH doomed AS (
    SELECT stale.message_id
    FROM chat_message_queue AS stale
    WHERE stale.message_timestamp <= sqlc.arg(cutoff)
      AND stale.completed_at IS NULL
      AND stale.failed_at IS NULL
      AND (stale.lease_until IS NULL OR stale.lease_until <= NOW())
    ORDER BY stale.message_timestamp, stale.message_id
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(batch_limit)::bigint
)
DELETE FROM chat_message_queue AS q
USING doomed
WHERE q.message_id = doomed.message_id;

-- name: ChatQueueDeleteCompleted :execrows
WITH doomed AS (
    SELECT completed.message_id
    FROM chat_message_queue AS completed
    WHERE completed.completed_at <= sqlc.arg(cutoff)::timestamptz
    ORDER BY completed.completed_at, completed.message_id
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(batch_limit)::bigint
)
DELETE FROM chat_message_queue AS q
USING doomed
WHERE q.message_id = doomed.message_id;

-- name: ChatQueueDeleteFailed :execrows
WITH doomed AS (
    SELECT failed.message_id
    FROM chat_message_queue AS failed
    WHERE failed.failed_at <= sqlc.arg(cutoff)::timestamptz
    ORDER BY failed.failed_at, failed.message_id
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(batch_limit)::bigint
)
DELETE FROM chat_message_queue AS q
USING doomed
WHERE q.message_id = doomed.message_id;
