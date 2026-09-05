package confimport

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/jackc/pgx/v5/pgtype"
)

type (
	//nolint:recvcheck // JSON encoding uses a value receiver while decoding must mutate a pointer.
	Channel dbsql.Channel
	//nolint:recvcheck // JSON encoding uses a value receiver while decoding must mutate a pointer.
	CommandInfo dbsql.CommandInfo
	//nolint:recvcheck // JSON encoding uses a value receiver while decoding must mutate a pointer.
	RepeatedCommand dbsql.RepeatedCommand
	//nolint:recvcheck // JSON encoding uses a value receiver while decoding must mutate a pointer.
	Autoreply        dbsql.Autoreply
	Quote            = dbsql.Quote
	CustomCommand    = dbsql.CustomCommand
	CommandList      = dbsql.CommandList
	ScheduledCommand = dbsql.ScheduledCommand
	Variable         = dbsql.Variable
)

func (c Channel) MarshalJSONTo(out *jsontext.Encoder) error {
	type alias Channel
	return marshalJSON(out, "channel", struct {
		alias
		Bullet   *string `json:"bullet"`
		Cooldown *int32  `json:"cooldown"`
	}{
		alias:    alias(c),
		Bullet:   nullStringPointer(c.Bullet),
		Cooldown: nullInt32Pointer(c.Cooldown),
	})
}

func (c *Channel) UnmarshalJSONFrom(in *jsontext.Decoder) error {
	type alias Channel
	value := struct {
		*alias
		Bullet   *string `json:"bullet"`
		Cooldown *int32  `json:"cooldown"`
	}{alias: (*alias)(c)}
	if err := unmarshalJSON(in, "channel", &value); err != nil {
		return err
	}
	c.Bullet = nullString(value.Bullet)
	c.Cooldown = nullInt32(value.Cooldown)
	return nil
}

func (c CommandInfo) MarshalJSONTo(out *jsontext.Encoder) error {
	type alias CommandInfo
	return marshalJSON(out, "command info", struct {
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

func (c *CommandInfo) UnmarshalJSONFrom(in *jsontext.Decoder) error {
	type alias CommandInfo
	value := struct {
		*alias
		LastUsed        *time.Time `json:"last_used"`
		CustomCommandID *int64     `json:"custom_command_id"`
		CommandListID   *int64     `json:"command_list_id"`
	}{alias: (*alias)(c)}
	if err := unmarshalJSON(in, "command info", &value); err != nil {
		return err
	}
	c.LastUsed = nullTime(value.LastUsed)
	c.CustomCommandID = nullInt64(value.CustomCommandID)
	c.CommandListID = nullInt64(value.CommandListID)
	return nil
}

func (r RepeatedCommand) MarshalJSONTo(out *jsontext.Encoder) error {
	type alias RepeatedCommand
	return marshalJSON(out, "repeated command", struct {
		alias
		InitTimestamp *time.Time `json:"init_timestamp"`
	}{
		alias:         alias(r),
		InitTimestamp: nullTimePointer(r.InitTimestamp),
	})
}

func (r *RepeatedCommand) UnmarshalJSONFrom(in *jsontext.Decoder) error {
	type alias RepeatedCommand
	value := struct {
		*alias
		InitTimestamp *time.Time `json:"init_timestamp"`
	}{alias: (*alias)(r)}
	if err := unmarshalJSON(in, "repeated command", &value); err != nil {
		return err
	}
	r.InitTimestamp = nullTime(value.InitTimestamp)
	return nil
}

func (a Autoreply) MarshalJSONTo(out *jsontext.Encoder) error {
	type alias Autoreply
	return marshalJSON(out, "autoreply", struct {
		alias
		OrigPattern *string `json:"orig_pattern"`
	}{
		alias:       alias(a),
		OrigPattern: nullStringPointer(a.OrigPattern),
	})
}

func (a *Autoreply) UnmarshalJSONFrom(in *jsontext.Decoder) error {
	type alias Autoreply
	value := struct {
		*alias
		OrigPattern *string `json:"orig_pattern"`
	}{alias: (*alias)(a)}
	if err := unmarshalJSON(in, "autoreply", &value); err != nil {
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

func marshalJSON(out *jsontext.Encoder, label string, value any) error {
	if err := json.MarshalEncode(out, value); err != nil {
		return fmt.Errorf("marshaling %s: %w", label, err)
	}
	return nil
}

func unmarshalJSON(in *jsontext.Decoder, label string, value any) error {
	if err := json.UnmarshalDecode(in, value); err != nil {
		return fmt.Errorf("unmarshaling %s: %w", label, err)
	}
	return nil
}
