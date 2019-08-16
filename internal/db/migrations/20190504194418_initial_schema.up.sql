-- This file is subject to change until the first real deployment of the bot.
-- Do not rely on these schema migrations remaining consistent until this
-- message has been removed.

BEGIN;


CREATE TYPE access_level AS ENUM (
    'everyone',
    'subscriber',
    'moderator',
    'broadcaster',
    'admin'
);


CREATE TABLE channels (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    created_at timestamptz DEFAULT NOW() NOT NULL,
    updated_at timestamptz DEFAULT NOW() NOT NULL,

    user_id bigint NOT NULL UNIQUE,
    name text NOT NULL,
    bot_name text NOT NULL,
    active boolean NOT NULL,
    prefix text NOT NULL,
    bullet text,

    mode access_level NOT NULL,

    ignored text[] DEFAULT '{}' NOT NULL,
    custom_owners text[] DEFAULT '{}' NOT NULL,
    custom_mods text[] DEFAULT '{}' NOT NULL,
    custom_regulars text[] DEFAULT '{}' NOT NULL,

    cooldown int,

    last_fm text NOT NULL,
    parse_youtube boolean NOT NULL,
    extra_life_id int NOT NULL,
    raffle_enabled boolean NOT NULL,
    steam_id text NOT NULL,
    urban_enabled boolean NOT NULL,

    roll_level access_level NOT NULL,
    roll_cooldown int NOT NULL,
    roll_default int NOT NULL,

    should_moderate boolean NOT NULL,
    display_warnings boolean NOT NULL,
    enable_warnings boolean NOT NULL,
    timeout_duration int NOT NULL,
    enable_filters boolean NOT NULL,

    filter_links boolean NOT NULL,
    permitted_links text[] DEFAULT '{}' NOT NULL,
    subs_may_link boolean NOT NULL,

    filter_caps boolean NOT NULL,
    filter_caps_min_chars int NOT NULL,
    filter_caps_percentage int NOT NULL,
    filter_caps_min_caps int NOT NULL,

    filter_emotes boolean NOT NULL,
    filter_emotes_max int NOT NULL,
    filter_emotes_single boolean NOT NULL,

    filter_symbols boolean NOT NULL,
    filter_symbols_percentage int NOT NULL,
    filter_symbols_min_symbols int NOT NULL,

    filter_me boolean NOT NULL,
    filter_max_length int NOT NULL,

    filter_banned_phrases boolean NOT NULL,
    filter_banned_phrases_patterns text[] DEFAULT '{}' NOT NULL,

    CHECK (prefix != ''),
    CHECK (filter_caps_percentage BETWEEN 0 and 100),
    CHECK (filter_symbols_percentage BETWEEN 0 and 100),
    CHECK (timeout_duration >= 0)
);

CREATE INDEX channels_user_id_idx on channels (user_id);


CREATE TABLE custom_commands (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    created_at timestamptz DEFAULT NOW() NOT NULL,
    updated_at timestamptz DEFAULT NOW() NOT NULL,

    channel_id bigint REFERENCES channels (id) NOT NULL,

    message text NOT NULL
);

CREATE INDEX custom_commands_channel_id_idx ON custom_commands (channel_id);


CREATE TABLE quotes (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    created_at timestamptz DEFAULT NOW() NOT NULL,
    updated_at timestamptz DEFAULT NOW() NOT NULL,

    channel_id bigint REFERENCES channels (id) NOT NULL,

    num int NOT NULL,
    quote text NOT NULL,

    creator text NOT NULL,
    editor text NOT NULL,

    UNIQUE (channel_id, num)
);

CREATE INDEX quotes_channel_id_num_idx ON quotes (channel_id, num);


CREATE TABLE autoreplies (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    created_at timestamptz DEFAULT NOW() NOT NULL,
    updated_at timestamptz DEFAULT NOW() NOT NULL,

    channel_id bigint REFERENCES channels (id) NOT NULL,

    num int NOT NULL,
    trigger text NOT NULL,
    orig_pattern text,
    response text NOT NULL,
    count int NOT NULL,

    creator text NOT NULL,
    editor text NOT NULL,

    UNIQUE (channel_id, num)
);

CREATE INDEX autoreplies_channel_id_idx ON autoreplies (channel_id);
CREATE INDEX autoreplies_num_idx ON autoreplies (num ASC);

CREATE TABLE variables (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    created_at timestamptz DEFAULT NOW() NOT NULL,
    updated_at timestamptz DEFAULT NOW() NOT NULL,

    channel_id bigint REFERENCES channels (id) NOT NULL,

    name text NOT NULL,
    value text NOT NULL,

    UNIQUE (channel_id, name)
);

CREATE INDEX variables_channel_id_name_idx ON variables (channel_id, name);


CREATE TABLE twitch_tokens (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    created_at timestamptz DEFAULT NOW() NOT NULL,
    updated_at timestamptz DEFAULT NOW() NOT NULL,

    twitch_id bigint NOT NULL UNIQUE,
    bot_name text UNIQUE,

    access_token text NOT NULL,
    token_type text NOT NULL,
    refresh_token text NOT NULL,
    expiry timestamptz NOT NULL
);

CREATE INDEX twitch_tokens_twitch_id_idx ON twitch_tokens (twitch_id);


CREATE TABLE blocked_users (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    created_at timestamptz DEFAULT NOW() NOT NULL,

    twitch_id bigint NOT NULL UNIQUE
);

CREATE TABLE command_lists (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    created_at timestamptz DEFAULT NOW() NOT NULL,
    updated_at timestamptz DEFAULT NOW() NOT NULL,

    channel_id bigint REFERENCES channels (id) NOT NULL,

    items text[] DEFAULT '{}' NOT NULL
);


CREATE TABLE command_infos (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    created_at timestamptz DEFAULT NOW() NOT NULL,
    updated_at timestamptz DEFAULT NOW() NOT NULL,

    channel_id bigint REFERENCES channels (id) NOT NULL,

    name text NOT NULL,
    access_level access_level NOT NULL,
    count bigint NOT NULL,
    creator text NOT NULL,
    editor text NOT NULL,
    last_used timestamptz,

    custom_command_id bigint REFERENCES custom_commands (id) UNIQUE,
    command_list_id bigint REFERENCES command_lists (id) UNIQUE,

    UNIQUE (channel_id, name),
    CONSTRAINT chk_unique_id CHECK (
        (
            (CASE WHEN custom_command_id IS NULL THEN 0 ELSE 1 end)
            + (CASE WHEN command_list_id IS NULL THEN 0 ELSE 1 end)
        ) = 1
    )
);

CREATE INDEX command_infos_channel_id_name_idx ON command_infos (channel_id, name);


CREATE TABLE repeated_commands (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    created_at timestamptz DEFAULT NOW() NOT NULL,
    updated_at timestamptz DEFAULT NOW() NOT NULL,

    channel_id bigint REFERENCES channels (id) NOT NULL,
    command_info_id bigint REFERENCES command_infos (id) NOT NULL UNIQUE,

    enabled boolean NOT NULL,
    delay int NOT NULL,
    message_diff bigint DEFAULT 1 NOT NULL,
    last_count bigint NOT NULL,

    creator text NOT NULL,
    editor text NOT NULL
);

CREATE INDEX repeated_commands_channel_id_idx ON repeated_commands (channel_id);
CREATE INDEX repeated_commands_command_info_id_idx ON repeated_commands (command_info_id);


CREATE TABLE scheduled_commands (
    id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    created_at timestamptz DEFAULT NOW() NOT NULL,
    updated_at timestamptz DEFAULT NOW() NOT NULL,

    channel_id bigint REFERENCES channels (id) NOT NULL,
    command_info_id bigint REFERENCES command_infos (id) NOT NULL UNIQUE,

    enabled boolean NOT NULL,
    cron_expression text NOT NULL,
    message_diff bigint DEFAULT 1 NOT NULL,
    last_count bigint NOT NULL,

    creator text NOT NULL,
    editor text NOT NULL
);

CREATE INDEX scheduled_commands_channel_id_idx ON scheduled_commands (channel_id);
CREATE INDEX scheduled_commands_command_info_id_idx ON scheduled_commands (command_info_id);


COMMIT;
