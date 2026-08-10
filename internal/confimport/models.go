package confimport

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/jackc/pgx/v5/pgtype"
)

type (
	Channel          dbsql.Channel
	CommandInfo      dbsql.CommandInfo
	RepeatedCommand  dbsql.RepeatedCommand
	Autoreply        dbsql.Autoreply
	Quote            = dbsql.Quote
	CustomCommand    = dbsql.CustomCommand
	CommandList      = dbsql.CommandList
	ScheduledCommand = dbsql.ScheduledCommand
	Variable         = dbsql.Variable
)

func (c Channel) MarshalJSON() ([]byte, error) {
	type alias Channel
	return marshalJSON("channel", struct {
		alias
		Bullet   *string `json:"bullet"`
		Cooldown *int32  `json:"cooldown"`
	}{
		alias:    alias(c),
		Bullet:   nullStringPointer(c.Bullet),
		Cooldown: nullInt32Pointer(c.Cooldown),
	})
}

func (c *Channel) UnmarshalJSON(data []byte) error {
	type alias Channel
	value := struct {
		*alias
		Bullet   *string `json:"bullet"`
		Cooldown *int32  `json:"cooldown"`
	}{alias: (*alias)(c)}
	if err := unmarshalJSON("channel", data, &value); err != nil {
		return err
	}
	c.Bullet = nullString(value.Bullet)
	c.Cooldown = nullInt32(value.Cooldown)
	return nil
}

func (c CommandInfo) MarshalJSON() ([]byte, error) {
	type alias CommandInfo
	return marshalJSON("command info", struct {
		alias
		LastUsed        *time.Time `json:"last_used"`
		CustomCommandID *int64     `json:"custom_command_id"`
		CommandListID   *int64     `json:"command_list_id"`
	}{
		alias:           alias(c),
		LastUsed:        nullTimePointer(c.LastUsed),
		CustomCommandID: nullInt64Pointer(c.CustomCommandID),
		CommandListID:   nullInt64Pointer(c.CommandListID),
	})
}

func (c *CommandInfo) UnmarshalJSON(data []byte) error {
	type alias CommandInfo
	value := struct {
		*alias
		LastUsed        *time.Time `json:"last_used"`
		CustomCommandID *int64     `json:"custom_command_id"`
		CommandListID   *int64     `json:"command_list_id"`
	}{alias: (*alias)(c)}
	if err := unmarshalJSON("command info", data, &value); err != nil {
		return err
	}
	c.LastUsed = nullTime(value.LastUsed)
	c.CustomCommandID = nullInt64(value.CustomCommandID)
	c.CommandListID = nullInt64(value.CommandListID)
	return nil
}

func (r RepeatedCommand) MarshalJSON() ([]byte, error) {
	type alias RepeatedCommand
	return marshalJSON("repeated command", struct {
		alias
		InitTimestamp *time.Time `json:"init_timestamp"`
	}{
		alias:         alias(r),
		InitTimestamp: nullTimePointer(r.InitTimestamp),
	})
}

func (r *RepeatedCommand) UnmarshalJSON(data []byte) error {
	type alias RepeatedCommand
	value := struct {
		*alias
		InitTimestamp *time.Time `json:"init_timestamp"`
	}{alias: (*alias)(r)}
	if err := unmarshalJSON("repeated command", data, &value); err != nil {
		return err
	}
	r.InitTimestamp = nullTime(value.InitTimestamp)
	return nil
}

func (a Autoreply) MarshalJSON() ([]byte, error) {
	type alias Autoreply
	return marshalJSON("autoreply", struct {
		alias
		OrigPattern *string `json:"orig_pattern"`
	}{
		alias:       alias(a),
		OrigPattern: nullStringPointer(a.OrigPattern),
	})
}

func (a *Autoreply) UnmarshalJSON(data []byte) error {
	type alias Autoreply
	value := struct {
		*alias
		OrigPattern *string `json:"orig_pattern"`
	}{alias: (*alias)(a)}
	if err := unmarshalJSON("autoreply", data, &value); err != nil {
		return err
	}
	a.OrigPattern = nullString(value.OrigPattern)
	return nil
}

func nullString(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return dbsql.TextFrom(*value)
}

func nullStringPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullInt32(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return dbsql.Int4From(*value)
}

func nullInt32Pointer(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	return &value.Int32
}

func nullInt64(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}
	return dbsql.Int8From(*value)
}

func nullInt64Pointer(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Int64
}

func nullTime(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return dbsql.TimestamptzFrom(*value)
}

func nullTimePointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func marshalJSON(label string, value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshaling %s: %w", label, err)
	}
	return data, nil
}

func unmarshalJSON(label string, data []byte, value any) error {
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("unmarshaling %s: %w", label, err)
	}
	return nil
}
