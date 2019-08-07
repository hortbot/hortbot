BEGIN;

DROP TABLE IF EXISTS
    channels,
    custom_commands,
    quotes,
    autoreplies,
    variables,
    twitch_tokens,
    blocked_users,
    command_lists,
    command_infos,
    repeated_commands,
    scheduled_commands
CASCADE;

DROP TYPE IF EXISTS access_level;

COMMIT;
