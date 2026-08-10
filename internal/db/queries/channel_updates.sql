-- name: UpdateChannelUserLists :exec
UPDATE channels
SET custom_owners = sqlc.arg(custom_owners)::text[],
    custom_mods = sqlc.arg(custom_mods)::text[],
    custom_regulars = sqlc.arg(custom_regulars)::text[],
    ignored = sqlc.arg(ignored)::text[],
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);

-- name: UpdateChannelRaffleEnabled :exec
UPDATE channels
SET raffle_enabled = sqlc.arg(raffle_enabled),
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);

-- name: UpdateChannelSettings :exec
UPDATE channels
SET prefix = sqlc.arg(prefix),
    bullet = sqlc.narg(bullet)::text,
    mode = sqlc.arg(mode),
    cooldown = sqlc.narg(cooldown)::integer,
    last_fm = sqlc.arg(last_fm),
    parse_youtube = sqlc.arg(parse_youtube),
    extra_life_id = sqlc.arg(extra_life_id),
    steam_id = sqlc.arg(steam_id),
    urban_enabled = sqlc.arg(urban_enabled),
    tweet = sqlc.arg(tweet),
    roll_level = sqlc.arg(roll_level),
    roll_cooldown = sqlc.arg(roll_cooldown),
    roll_default = sqlc.arg(roll_default),
    should_moderate = sqlc.arg(should_moderate),
    display_warnings = sqlc.arg(display_warnings),
    enable_warnings = sqlc.arg(enable_warnings),
    timeout_duration = sqlc.arg(timeout_duration),
    enable_filters = sqlc.arg(enable_filters),
    filter_links = sqlc.arg(filter_links),
    permitted_links = sqlc.arg(permitted_links)::text[],
    subs_may_link = sqlc.arg(subs_may_link),
    filter_caps = sqlc.arg(filter_caps),
    filter_caps_min_chars = sqlc.arg(filter_caps_min_chars),
    filter_caps_percentage = sqlc.arg(filter_caps_percentage),
    filter_caps_min_caps = sqlc.arg(filter_caps_min_caps),
    filter_emotes = sqlc.arg(filter_emotes),
    filter_emotes_max = sqlc.arg(filter_emotes_max),
    filter_emotes_single = sqlc.arg(filter_emotes_single),
    filter_symbols = sqlc.arg(filter_symbols),
    filter_symbols_percentage = sqlc.arg(filter_symbols_percentage),
    filter_symbols_min_symbols = sqlc.arg(filter_symbols_min_symbols),
    filter_me = sqlc.arg(filter_me),
    filter_max_length = sqlc.arg(filter_max_length),
    filter_banned_phrases = sqlc.arg(filter_banned_phrases),
    filter_banned_phrases_patterns = sqlc.arg(filter_banned_phrases_patterns)::text[],
    filter_exempt_level = sqlc.arg(filter_exempt_level),
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id);
