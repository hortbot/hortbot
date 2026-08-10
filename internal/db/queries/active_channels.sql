-- name: ListActiveChannelAssignments :many
SELECT c.twitch_id, c.bot_name
FROM channels c
LEFT JOIN twitch_tokens tt ON tt.twitch_id = c.twitch_id
LEFT JOIN moderated_channels m
    ON m.broadcaster_id = c.twitch_id
   AND m.bot_name = c.bot_name
WHERE c.active
  AND ('channel:bot' = ANY(tt.scopes) OR m.id IS NOT NULL);

-- name: ListPublicActiveChannels :many
SELECT c.name, c.display_name
FROM channels c
LEFT JOIN twitch_tokens tt ON tt.twitch_id = c.twitch_id
LEFT JOIN moderated_channels m
    ON m.broadcaster_id = c.twitch_id
   AND m.bot_name = c.bot_name
WHERE c.active
  AND ('channel:bot' = ANY(tt.scopes) OR m.id IS NOT NULL)
ORDER BY c.name;

-- name: CountActiveChannelAssignments :one
SELECT COUNT(*)::integer AS channel_count,
       COUNT(DISTINCT c.bot_name)::integer AS bot_count
FROM channels c
LEFT JOIN twitch_tokens tt ON tt.twitch_id = c.twitch_id
LEFT JOIN moderated_channels m
    ON m.broadcaster_id = c.twitch_id
   AND m.bot_name = c.bot_name
WHERE c.active
  AND ('channel:bot' = ANY(tt.scopes) OR m.id IS NOT NULL);
