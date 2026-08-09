BEGIN;

CREATE TABLE chat_message_queue_keys (
    broadcaster_login text PRIMARY KEY,
    lease_token text,
    lease_until timestamptz,

    CHECK ((lease_token IS NULL) = (lease_until IS NULL))
);

CREATE TABLE chat_message_queue (
    message_id text PRIMARY KEY,
    broadcaster_login text NOT NULL REFERENCES chat_message_queue_keys (broadcaster_login),
    message_timestamp timestamptz NOT NULL,
    enqueued_at timestamptz NOT NULL,
    payload jsonb NOT NULL,
    lease_token text,
    lease_until timestamptz,
    completed_at timestamptz,
    failed_at timestamptz,
    last_error text,

    CHECK ((lease_token IS NULL) = (lease_until IS NULL)),
    CHECK ((failed_at IS NULL) = (last_error IS NULL)),
    CHECK (completed_at IS NULL OR failed_at IS NULL)
);

CREATE INDEX chat_message_queue_claim_idx
    ON chat_message_queue (enqueued_at)
    WHERE completed_at IS NULL AND failed_at IS NULL;
CREATE INDEX chat_message_queue_stale_idx
    ON chat_message_queue (message_timestamp)
    WHERE completed_at IS NULL AND failed_at IS NULL;
CREATE INDEX chat_message_queue_completed_at_idx
    ON chat_message_queue (completed_at)
    WHERE completed_at IS NOT NULL;
CREATE INDEX chat_message_queue_failed_at_idx
    ON chat_message_queue (failed_at)
    WHERE failed_at IS NOT NULL;

CREATE TABLE eventsub_sync_requests (
    singleton boolean PRIMARY KEY DEFAULT TRUE,
    version bigint DEFAULT 0 NOT NULL,

    CHECK (singleton)
);

INSERT INTO eventsub_sync_requests DEFAULT VALUES;

COMMIT;
