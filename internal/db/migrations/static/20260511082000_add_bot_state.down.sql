BEGIN;

DROP TABLE IF EXISTS bot_action_usage_stats;
DROP TABLE IF EXISTS bot_builtin_usage_stats;
DROP TABLE IF EXISTS bot_raffle_entries;
DROP TABLE IF EXISTS web_auth_states;
DROP TABLE IF EXISTS bot_filter_warnings;
DROP TABLE IF EXISTS bot_confirmations;
DROP TABLE IF EXISTS bot_link_permits;
DROP TABLE IF EXISTS bot_autoreply_cooldowns;
DROP TABLE IF EXISTS bot_scheduled_command_cooldowns;
DROP TABLE IF EXISTS bot_repeat_cooldowns;
DROP TABLE IF EXISTS bot_command_cooldowns;

COMMIT;
