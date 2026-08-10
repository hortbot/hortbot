-- name: InsertImportedChannel :one
INSERT INTO channels OVERRIDING USER VALUE
SELECT (jsonb_populate_record(NULL::channels, sqlc.arg(data)::jsonb)).*
RETURNING id;

-- name: InsertFixtureChannel :one
INSERT INTO channels
SELECT (jsonb_populate_record(NULL::channels, sqlc.arg(data)::jsonb)).*
RETURNING id;

-- name: InsertFixtureCustomCommand :one
INSERT INTO custom_commands
SELECT (jsonb_populate_record(NULL::custom_commands, sqlc.arg(data)::jsonb)).*
RETURNING id;

-- name: InsertFixtureCommandInfo :one
INSERT INTO command_infos
SELECT (jsonb_populate_record(NULL::command_infos, sqlc.arg(data)::jsonb)).*
RETURNING id;

-- name: InsertFixtureRepeatedCommand :one
INSERT INTO repeated_commands
SELECT (jsonb_populate_record(NULL::repeated_commands, sqlc.arg(data)::jsonb)).*
RETURNING id;

-- name: InsertFixtureScheduledCommand :one
INSERT INTO scheduled_commands
SELECT (jsonb_populate_record(NULL::scheduled_commands, sqlc.arg(data)::jsonb)).*
RETURNING id;

-- name: InsertImportedQuote :one
INSERT INTO quotes OVERRIDING USER VALUE
SELECT (jsonb_populate_record(NULL::quotes, sqlc.arg(data)::jsonb)).*
RETURNING id;

-- name: InsertImportedCustomCommand :one
INSERT INTO custom_commands OVERRIDING USER VALUE
SELECT (jsonb_populate_record(NULL::custom_commands, sqlc.arg(data)::jsonb)).*
RETURNING id;

-- name: InsertImportedCommandList :one
INSERT INTO command_lists OVERRIDING USER VALUE
SELECT (jsonb_populate_record(NULL::command_lists, sqlc.arg(data)::jsonb)).*
RETURNING id;

-- name: InsertImportedCommandInfo :one
INSERT INTO command_infos OVERRIDING USER VALUE
SELECT (jsonb_populate_record(NULL::command_infos, sqlc.arg(data)::jsonb)).*
RETURNING id;

-- name: InsertImportedRepeatedCommand :one
INSERT INTO repeated_commands OVERRIDING USER VALUE
SELECT (jsonb_populate_record(NULL::repeated_commands, sqlc.arg(data)::jsonb)).*
RETURNING id;

-- name: InsertImportedScheduledCommand :one
INSERT INTO scheduled_commands OVERRIDING USER VALUE
SELECT (jsonb_populate_record(NULL::scheduled_commands, sqlc.arg(data)::jsonb)).*
RETURNING id;

-- name: InsertImportedAutoreply :one
INSERT INTO autoreplies OVERRIDING USER VALUE
SELECT (jsonb_populate_record(NULL::autoreplies, sqlc.arg(data)::jsonb)).*
RETURNING id;

-- name: InsertImportedVariable :one
INSERT INTO variables OVERRIDING USER VALUE
SELECT (jsonb_populate_record(NULL::variables, sqlc.arg(data)::jsonb)).*
RETURNING id;
