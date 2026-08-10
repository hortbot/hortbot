-- name: RequestEventsubSync :exec
UPDATE eventsub_sync_requests
SET version = version + 1
WHERE singleton;

-- name: GetEventsubSyncVersion :one
SELECT version
FROM eventsub_sync_requests
WHERE singleton;
