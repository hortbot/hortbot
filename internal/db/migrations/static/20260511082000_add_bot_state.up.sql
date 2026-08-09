BEGIN;

CREATE TABLE bot_command_cooldowns (
    channel text NOT NULL,
    command_key text NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (channel, command_key)
);
CREATE INDEX bot_command_cooldowns_expires_at_idx ON bot_command_cooldowns (expires_at);

CREATE TABLE bot_repeat_cooldowns (
    channel text NOT NULL,
    repeated_command_id bigint NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (channel, repeated_command_id)
);
CREATE INDEX bot_repeat_cooldowns_expires_at_idx ON bot_repeat_cooldowns (expires_at);

CREATE TABLE bot_scheduled_command_cooldowns (
    channel text NOT NULL,
    scheduled_command_id bigint NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (channel, scheduled_command_id)
);
CREATE INDEX bot_scheduled_command_cooldowns_expires_at_idx ON bot_scheduled_command_cooldowns (expires_at);

CREATE TABLE bot_autoreply_cooldowns (
    channel text NOT NULL,
    autoreply_id bigint NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (channel, autoreply_id)
);
CREATE INDEX bot_autoreply_cooldowns_expires_at_idx ON bot_autoreply_cooldowns (expires_at);

CREATE TABLE bot_link_permits (
    channel text NOT NULL,
    user_id text NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (channel, user_id)
);
CREATE INDEX bot_link_permits_expires_at_idx ON bot_link_permits (expires_at);

CREATE TABLE bot_confirmations (
    channel text NOT NULL,
    user_id text NOT NULL,
    confirmation_key text NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (channel, user_id, confirmation_key)
);
CREATE INDEX bot_confirmations_expires_at_idx ON bot_confirmations (expires_at);

CREATE TABLE bot_filter_warnings (
    channel text NOT NULL,
    user_id text NOT NULL,
    filter_name text NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (channel, user_id, filter_name)
);
CREATE INDEX bot_filter_warnings_expires_at_idx ON bot_filter_warnings (expires_at);

CREATE TABLE web_auth_states (
    key text PRIMARY KEY,
    value bytea NOT NULL,
    expires_at timestamptz NOT NULL
);
CREATE INDEX web_auth_states_expires_at_idx ON web_auth_states (expires_at);

CREATE TABLE bot_raffle_entries (
    channel text NOT NULL,
    user_id text NOT NULL,
    PRIMARY KEY (channel, user_id)
);

CREATE TABLE bot_builtin_usage_stats (
    name text PRIMARY KEY,
    count bigint NOT NULL DEFAULT 0
);

CREATE TABLE bot_action_usage_stats (
    name text PRIMARY KEY,
    count bigint NOT NULL DEFAULT 0
);

COMMIT;
