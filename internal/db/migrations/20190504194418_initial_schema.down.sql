BEGIN;

DROP TABLE IF EXISTS
    channels,
    custom_commands,
    quotes,
    repeated_commands,
    scheduled_commands,
    autoreplies,
    variables,
    twitch_tokens
CASCADE;

DROP TYPE IF EXISTS access_level;

COMMIT;
