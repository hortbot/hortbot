BEGIN;

DROP TABLE IF EXISTS chat_message_queue CASCADE;
DROP TABLE IF EXISTS chat_message_queue_keys CASCADE;
DROP TABLE IF EXISTS eventsub_sync_requests CASCADE;

COMMIT;
