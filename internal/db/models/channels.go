// Code generated by SQLBoiler (https://github.com/volatiletech/sqlboiler). DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"github.com/volatiletech/sqlboiler/queries/qmhelper"
	"github.com/volatiletech/sqlboiler/strmangle"
	"github.com/volatiletech/sqlboiler/types"
)

// Channel is an object representing the database table.
type Channel struct {
	ID             int64             `boil:"id" json:"id" toml:"id" yaml:"id"`
	CreatedAt      time.Time         `boil:"created_at" json:"created_at" toml:"created_at" yaml:"created_at"`
	UpdatedAt      time.Time         `boil:"updated_at" json:"updated_at" toml:"updated_at" yaml:"updated_at"`
	UserID         int64             `boil:"user_id" json:"user_id" toml:"user_id" yaml:"user_id"`
	Name           string            `boil:"name" json:"name" toml:"name" yaml:"name"`
	BotName        string            `boil:"bot_name" json:"bot_name" toml:"bot_name" yaml:"bot_name"`
	Active         bool              `boil:"active" json:"active" toml:"active" yaml:"active"`
	Prefix         string            `boil:"prefix" json:"prefix" toml:"prefix" yaml:"prefix"`
	Bullet         null.String       `boil:"bullet" json:"bullet,omitempty" toml:"bullet" yaml:"bullet,omitempty"`
	Ignored        types.StringArray `boil:"ignored" json:"ignored" toml:"ignored" yaml:"ignored"`
	CustomOwners   types.StringArray `boil:"custom_owners" json:"custom_owners" toml:"custom_owners" yaml:"custom_owners"`
	CustomMods     types.StringArray `boil:"custom_mods" json:"custom_mods" toml:"custom_mods" yaml:"custom_mods"`
	CustomRegulars types.StringArray `boil:"custom_regulars" json:"custom_regulars" toml:"custom_regulars" yaml:"custom_regulars"`
	Cooldown       null.Int          `boil:"cooldown" json:"cooldown,omitempty" toml:"cooldown" yaml:"cooldown,omitempty"`
	LastCommandAt  time.Time         `boil:"last_command_at" json:"last_command_at" toml:"last_command_at" yaml:"last_command_at"`

	R *channelR `boil:"-" json:"-" toml:"-" yaml:"-"`
	L channelL  `boil:"-" json:"-" toml:"-" yaml:"-"`
}

var ChannelColumns = struct {
	ID             string
	CreatedAt      string
	UpdatedAt      string
	UserID         string
	Name           string
	BotName        string
	Active         string
	Prefix         string
	Bullet         string
	Ignored        string
	CustomOwners   string
	CustomMods     string
	CustomRegulars string
	Cooldown       string
	LastCommandAt  string
}{
	ID:             "id",
	CreatedAt:      "created_at",
	UpdatedAt:      "updated_at",
	UserID:         "user_id",
	Name:           "name",
	BotName:        "bot_name",
	Active:         "active",
	Prefix:         "prefix",
	Bullet:         "bullet",
	Ignored:        "ignored",
	CustomOwners:   "custom_owners",
	CustomMods:     "custom_mods",
	CustomRegulars: "custom_regulars",
	Cooldown:       "cooldown",
	LastCommandAt:  "last_command_at",
}

// Generated where

type whereHelperint64 struct{ field string }

func (w whereHelperint64) EQ(x int64) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.EQ, x) }
func (w whereHelperint64) NEQ(x int64) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.NEQ, x) }
func (w whereHelperint64) LT(x int64) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.LT, x) }
func (w whereHelperint64) LTE(x int64) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.LTE, x) }
func (w whereHelperint64) GT(x int64) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.GT, x) }
func (w whereHelperint64) GTE(x int64) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.GTE, x) }

type whereHelpertime_Time struct{ field string }

func (w whereHelpertime_Time) EQ(x time.Time) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.EQ, x)
}
func (w whereHelpertime_Time) NEQ(x time.Time) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.NEQ, x)
}
func (w whereHelpertime_Time) LT(x time.Time) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LT, x)
}
func (w whereHelpertime_Time) LTE(x time.Time) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LTE, x)
}
func (w whereHelpertime_Time) GT(x time.Time) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GT, x)
}
func (w whereHelpertime_Time) GTE(x time.Time) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GTE, x)
}

type whereHelperstring struct{ field string }

func (w whereHelperstring) EQ(x string) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.EQ, x) }
func (w whereHelperstring) NEQ(x string) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.NEQ, x) }
func (w whereHelperstring) LT(x string) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.LT, x) }
func (w whereHelperstring) LTE(x string) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.LTE, x) }
func (w whereHelperstring) GT(x string) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.GT, x) }
func (w whereHelperstring) GTE(x string) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.GTE, x) }

type whereHelperbool struct{ field string }

func (w whereHelperbool) EQ(x bool) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.EQ, x) }
func (w whereHelperbool) NEQ(x bool) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.NEQ, x) }
func (w whereHelperbool) LT(x bool) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.LT, x) }
func (w whereHelperbool) LTE(x bool) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.LTE, x) }
func (w whereHelperbool) GT(x bool) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.GT, x) }
func (w whereHelperbool) GTE(x bool) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.GTE, x) }

type whereHelpernull_String struct{ field string }

func (w whereHelpernull_String) EQ(x null.String) qm.QueryMod {
	return qmhelper.WhereNullEQ(w.field, false, x)
}
func (w whereHelpernull_String) NEQ(x null.String) qm.QueryMod {
	return qmhelper.WhereNullEQ(w.field, true, x)
}
func (w whereHelpernull_String) IsNull() qm.QueryMod    { return qmhelper.WhereIsNull(w.field) }
func (w whereHelpernull_String) IsNotNull() qm.QueryMod { return qmhelper.WhereIsNotNull(w.field) }
func (w whereHelpernull_String) LT(x null.String) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LT, x)
}
func (w whereHelpernull_String) LTE(x null.String) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LTE, x)
}
func (w whereHelpernull_String) GT(x null.String) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GT, x)
}
func (w whereHelpernull_String) GTE(x null.String) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GTE, x)
}

type whereHelpertypes_StringArray struct{ field string }

func (w whereHelpertypes_StringArray) EQ(x types.StringArray) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.EQ, x)
}
func (w whereHelpertypes_StringArray) NEQ(x types.StringArray) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.NEQ, x)
}
func (w whereHelpertypes_StringArray) LT(x types.StringArray) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LT, x)
}
func (w whereHelpertypes_StringArray) LTE(x types.StringArray) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LTE, x)
}
func (w whereHelpertypes_StringArray) GT(x types.StringArray) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GT, x)
}
func (w whereHelpertypes_StringArray) GTE(x types.StringArray) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GTE, x)
}

type whereHelpernull_Int struct{ field string }

func (w whereHelpernull_Int) EQ(x null.Int) qm.QueryMod {
	return qmhelper.WhereNullEQ(w.field, false, x)
}
func (w whereHelpernull_Int) NEQ(x null.Int) qm.QueryMod {
	return qmhelper.WhereNullEQ(w.field, true, x)
}
func (w whereHelpernull_Int) IsNull() qm.QueryMod    { return qmhelper.WhereIsNull(w.field) }
func (w whereHelpernull_Int) IsNotNull() qm.QueryMod { return qmhelper.WhereIsNotNull(w.field) }
func (w whereHelpernull_Int) LT(x null.Int) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LT, x)
}
func (w whereHelpernull_Int) LTE(x null.Int) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LTE, x)
}
func (w whereHelpernull_Int) GT(x null.Int) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GT, x)
}
func (w whereHelpernull_Int) GTE(x null.Int) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GTE, x)
}

var ChannelWhere = struct {
	ID             whereHelperint64
	CreatedAt      whereHelpertime_Time
	UpdatedAt      whereHelpertime_Time
	UserID         whereHelperint64
	Name           whereHelperstring
	BotName        whereHelperstring
	Active         whereHelperbool
	Prefix         whereHelperstring
	Bullet         whereHelpernull_String
	Ignored        whereHelpertypes_StringArray
	CustomOwners   whereHelpertypes_StringArray
	CustomMods     whereHelpertypes_StringArray
	CustomRegulars whereHelpertypes_StringArray
	Cooldown       whereHelpernull_Int
	LastCommandAt  whereHelpertime_Time
}{
	ID:             whereHelperint64{field: "\"channels\".\"id\""},
	CreatedAt:      whereHelpertime_Time{field: "\"channels\".\"created_at\""},
	UpdatedAt:      whereHelpertime_Time{field: "\"channels\".\"updated_at\""},
	UserID:         whereHelperint64{field: "\"channels\".\"user_id\""},
	Name:           whereHelperstring{field: "\"channels\".\"name\""},
	BotName:        whereHelperstring{field: "\"channels\".\"bot_name\""},
	Active:         whereHelperbool{field: "\"channels\".\"active\""},
	Prefix:         whereHelperstring{field: "\"channels\".\"prefix\""},
	Bullet:         whereHelpernull_String{field: "\"channels\".\"bullet\""},
	Ignored:        whereHelpertypes_StringArray{field: "\"channels\".\"ignored\""},
	CustomOwners:   whereHelpertypes_StringArray{field: "\"channels\".\"custom_owners\""},
	CustomMods:     whereHelpertypes_StringArray{field: "\"channels\".\"custom_mods\""},
	CustomRegulars: whereHelpertypes_StringArray{field: "\"channels\".\"custom_regulars\""},
	Cooldown:       whereHelpernull_Int{field: "\"channels\".\"cooldown\""},
	LastCommandAt:  whereHelpertime_Time{field: "\"channels\".\"last_command_at\""},
}

// ChannelRels is where relationship names are stored.
var ChannelRels = struct {
	Quotes         string
	SimpleCommands string
}{
	Quotes:         "Quotes",
	SimpleCommands: "SimpleCommands",
}

// channelR is where relationships are stored.
type channelR struct {
	Quotes         QuoteSlice
	SimpleCommands SimpleCommandSlice
}

// NewStruct creates a new relationship struct
func (*channelR) NewStruct() *channelR {
	return &channelR{}
}

// channelL is where Load methods for each relationship are stored.
type channelL struct{}

var (
	channelAllColumns            = []string{"id", "created_at", "updated_at", "user_id", "name", "bot_name", "active", "prefix", "bullet", "ignored", "custom_owners", "custom_mods", "custom_regulars", "cooldown", "last_command_at"}
	channelColumnsWithoutDefault = []string{"user_id", "name", "bot_name", "active", "prefix", "bullet", "cooldown", "last_command_at"}
	channelColumnsWithDefault    = []string{"id", "created_at", "updated_at", "ignored", "custom_owners", "custom_mods", "custom_regulars"}
	channelPrimaryKeyColumns     = []string{"id"}
)

type (
	// ChannelSlice is an alias for a slice of pointers to Channel.
	// This should generally be used opposed to []Channel.
	ChannelSlice []*Channel

	channelQuery struct {
		*queries.Query
	}
)

// Cache for insert, update and upsert
var (
	channelType                 = reflect.TypeOf(&Channel{})
	channelMapping              = queries.MakeStructMapping(channelType)
	channelPrimaryKeyMapping, _ = queries.BindMapping(channelType, channelMapping, channelPrimaryKeyColumns)
	channelInsertCacheMut       sync.RWMutex
	channelInsertCache          = make(map[string]insertCache)
	channelUpdateCacheMut       sync.RWMutex
	channelUpdateCache          = make(map[string]updateCache)
	channelUpsertCacheMut       sync.RWMutex
	channelUpsertCache          = make(map[string]insertCache)
)

var (
	// Force time package dependency for automated UpdatedAt/CreatedAt.
	_ = time.Second
	// Force qmhelper dependency for where clause generation (which doesn't
	// always happen)
	_ = qmhelper.Where
)

// One returns a single channel record from the query.
func (q channelQuery) One(ctx context.Context, exec boil.ContextExecutor) (*Channel, error) {
	o := &Channel{}

	queries.SetLimit(q.Query, 1)

	err := q.Bind(ctx, exec, o)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: failed to execute a one query for channels")
	}

	return o, nil
}

// All returns all Channel records from the query.
func (q channelQuery) All(ctx context.Context, exec boil.ContextExecutor) (ChannelSlice, error) {
	var o []*Channel

	err := q.Bind(ctx, exec, &o)
	if err != nil {
		return nil, errors.Wrap(err, "models: failed to assign all query results to Channel slice")
	}

	return o, nil
}

// Count returns the count of all Channel records in the query.
func (q channelQuery) Count(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to count channels rows")
	}

	return count, nil
}

// Exists checks if the row exists in the table.
func (q channelQuery) Exists(ctx context.Context, exec boil.ContextExecutor) (bool, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)
	queries.SetLimit(q.Query, 1)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "models: failed to check if channels exists")
	}

	return count > 0, nil
}

// Quotes retrieves all the quote's Quotes with an executor.
func (o *Channel) Quotes(mods ...qm.QueryMod) quoteQuery {
	var queryMods []qm.QueryMod
	if len(mods) != 0 {
		queryMods = append(queryMods, mods...)
	}

	queryMods = append(queryMods,
		qm.Where("\"quotes\".\"channel_id\"=?", o.ID),
	)

	query := Quotes(queryMods...)
	queries.SetFrom(query.Query, "\"quotes\"")

	if len(queries.GetSelect(query.Query)) == 0 {
		queries.SetSelect(query.Query, []string{"\"quotes\".*"})
	}

	return query
}

// SimpleCommands retrieves all the simple_command's SimpleCommands with an executor.
func (o *Channel) SimpleCommands(mods ...qm.QueryMod) simpleCommandQuery {
	var queryMods []qm.QueryMod
	if len(mods) != 0 {
		queryMods = append(queryMods, mods...)
	}

	queryMods = append(queryMods,
		qm.Where("\"simple_commands\".\"channel_id\"=?", o.ID),
	)

	query := SimpleCommands(queryMods...)
	queries.SetFrom(query.Query, "\"simple_commands\"")

	if len(queries.GetSelect(query.Query)) == 0 {
		queries.SetSelect(query.Query, []string{"\"simple_commands\".*"})
	}

	return query
}

// LoadQuotes allows an eager lookup of values, cached into the
// loaded structs of the objects. This is for a 1-M or N-M relationship.
func (channelL) LoadQuotes(ctx context.Context, e boil.ContextExecutor, singular bool, maybeChannel interface{}, mods queries.Applicator) error {
	var slice []*Channel
	var object *Channel

	if singular {
		object = maybeChannel.(*Channel)
	} else {
		slice = *maybeChannel.(*[]*Channel)
	}

	args := make([]interface{}, 0, 1)
	if singular {
		if object.R == nil {
			object.R = &channelR{}
		}
		args = append(args, object.ID)
	} else {
	Outer:
		for _, obj := range slice {
			if obj.R == nil {
				obj.R = &channelR{}
			}

			for _, a := range args {
				if a == obj.ID {
					continue Outer
				}
			}

			args = append(args, obj.ID)
		}
	}

	if len(args) == 0 {
		return nil
	}

	query := NewQuery(qm.From(`quotes`), qm.WhereIn(`channel_id in ?`, args...))
	if mods != nil {
		mods.Apply(query)
	}

	results, err := query.QueryContext(ctx, e)
	if err != nil {
		return errors.Wrap(err, "failed to eager load quotes")
	}

	var resultSlice []*Quote
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice quotes")
	}

	if err = results.Close(); err != nil {
		return errors.Wrap(err, "failed to close results in eager load on quotes")
	}
	if err = results.Err(); err != nil {
		return errors.Wrap(err, "error occurred during iteration of eager loaded relations for quotes")
	}

	if singular {
		object.R.Quotes = resultSlice
		for _, foreign := range resultSlice {
			if foreign.R == nil {
				foreign.R = &quoteR{}
			}
			foreign.R.Channel = object
		}
		return nil
	}

	for _, foreign := range resultSlice {
		for _, local := range slice {
			if local.ID == foreign.ChannelID {
				local.R.Quotes = append(local.R.Quotes, foreign)
				if foreign.R == nil {
					foreign.R = &quoteR{}
				}
				foreign.R.Channel = local
				break
			}
		}
	}

	return nil
}

// LoadSimpleCommands allows an eager lookup of values, cached into the
// loaded structs of the objects. This is for a 1-M or N-M relationship.
func (channelL) LoadSimpleCommands(ctx context.Context, e boil.ContextExecutor, singular bool, maybeChannel interface{}, mods queries.Applicator) error {
	var slice []*Channel
	var object *Channel

	if singular {
		object = maybeChannel.(*Channel)
	} else {
		slice = *maybeChannel.(*[]*Channel)
	}

	args := make([]interface{}, 0, 1)
	if singular {
		if object.R == nil {
			object.R = &channelR{}
		}
		args = append(args, object.ID)
	} else {
	Outer:
		for _, obj := range slice {
			if obj.R == nil {
				obj.R = &channelR{}
			}

			for _, a := range args {
				if a == obj.ID {
					continue Outer
				}
			}

			args = append(args, obj.ID)
		}
	}

	if len(args) == 0 {
		return nil
	}

	query := NewQuery(qm.From(`simple_commands`), qm.WhereIn(`channel_id in ?`, args...))
	if mods != nil {
		mods.Apply(query)
	}

	results, err := query.QueryContext(ctx, e)
	if err != nil {
		return errors.Wrap(err, "failed to eager load simple_commands")
	}

	var resultSlice []*SimpleCommand
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice simple_commands")
	}

	if err = results.Close(); err != nil {
		return errors.Wrap(err, "failed to close results in eager load on simple_commands")
	}
	if err = results.Err(); err != nil {
		return errors.Wrap(err, "error occurred during iteration of eager loaded relations for simple_commands")
	}

	if singular {
		object.R.SimpleCommands = resultSlice
		for _, foreign := range resultSlice {
			if foreign.R == nil {
				foreign.R = &simpleCommandR{}
			}
			foreign.R.Channel = object
		}
		return nil
	}

	for _, foreign := range resultSlice {
		for _, local := range slice {
			if local.ID == foreign.ChannelID {
				local.R.SimpleCommands = append(local.R.SimpleCommands, foreign)
				if foreign.R == nil {
					foreign.R = &simpleCommandR{}
				}
				foreign.R.Channel = local
				break
			}
		}
	}

	return nil
}

// AddQuotes adds the given related objects to the existing relationships
// of the channel, optionally inserting them as new records.
// Appends related to o.R.Quotes.
// Sets related.R.Channel appropriately.
func (o *Channel) AddQuotes(ctx context.Context, exec boil.ContextExecutor, insert bool, related ...*Quote) error {
	var err error
	for _, rel := range related {
		if insert {
			rel.ChannelID = o.ID
			if err = rel.Insert(ctx, exec, boil.Infer()); err != nil {
				return errors.Wrap(err, "failed to insert into foreign table")
			}
		} else {
			updateQuery := fmt.Sprintf(
				"UPDATE \"quotes\" SET %s WHERE %s",
				strmangle.SetParamNames("\"", "\"", 1, []string{"channel_id"}),
				strmangle.WhereClause("\"", "\"", 2, quotePrimaryKeyColumns),
			)
			values := []interface{}{o.ID, rel.ID}

			if boil.DebugMode {
				fmt.Fprintln(boil.DebugWriter, updateQuery)
				fmt.Fprintln(boil.DebugWriter, values)
			}

			if _, err = exec.ExecContext(ctx, updateQuery, values...); err != nil {
				return errors.Wrap(err, "failed to update foreign table")
			}

			rel.ChannelID = o.ID
		}
	}

	if o.R == nil {
		o.R = &channelR{
			Quotes: related,
		}
	} else {
		o.R.Quotes = append(o.R.Quotes, related...)
	}

	for _, rel := range related {
		if rel.R == nil {
			rel.R = &quoteR{
				Channel: o,
			}
		} else {
			rel.R.Channel = o
		}
	}
	return nil
}

// AddSimpleCommands adds the given related objects to the existing relationships
// of the channel, optionally inserting them as new records.
// Appends related to o.R.SimpleCommands.
// Sets related.R.Channel appropriately.
func (o *Channel) AddSimpleCommands(ctx context.Context, exec boil.ContextExecutor, insert bool, related ...*SimpleCommand) error {
	var err error
	for _, rel := range related {
		if insert {
			rel.ChannelID = o.ID
			if err = rel.Insert(ctx, exec, boil.Infer()); err != nil {
				return errors.Wrap(err, "failed to insert into foreign table")
			}
		} else {
			updateQuery := fmt.Sprintf(
				"UPDATE \"simple_commands\" SET %s WHERE %s",
				strmangle.SetParamNames("\"", "\"", 1, []string{"channel_id"}),
				strmangle.WhereClause("\"", "\"", 2, simpleCommandPrimaryKeyColumns),
			)
			values := []interface{}{o.ID, rel.ID}

			if boil.DebugMode {
				fmt.Fprintln(boil.DebugWriter, updateQuery)
				fmt.Fprintln(boil.DebugWriter, values)
			}

			if _, err = exec.ExecContext(ctx, updateQuery, values...); err != nil {
				return errors.Wrap(err, "failed to update foreign table")
			}

			rel.ChannelID = o.ID
		}
	}

	if o.R == nil {
		o.R = &channelR{
			SimpleCommands: related,
		}
	} else {
		o.R.SimpleCommands = append(o.R.SimpleCommands, related...)
	}

	for _, rel := range related {
		if rel.R == nil {
			rel.R = &simpleCommandR{
				Channel: o,
			}
		} else {
			rel.R.Channel = o
		}
	}
	return nil
}

// Channels retrieves all the records using an executor.
func Channels(mods ...qm.QueryMod) channelQuery {
	mods = append(mods, qm.From("\"channels\""))
	return channelQuery{NewQuery(mods...)}
}

// FindChannel retrieves a single record by ID with an executor.
// If selectCols is empty Find will return all columns.
func FindChannel(ctx context.Context, exec boil.ContextExecutor, iD int64, selectCols ...string) (*Channel, error) {
	channelObj := &Channel{}

	sel := "*"
	if len(selectCols) > 0 {
		sel = strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, selectCols), ",")
	}
	query := fmt.Sprintf(
		"select %s from \"channels\" where \"id\"=$1", sel,
	)

	q := queries.Raw(query, iD)

	err := q.Bind(ctx, exec, channelObj)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: unable to select from channels")
	}

	return channelObj, nil
}

// Insert a single record using an executor.
// See boil.Columns.InsertColumnSet documentation to understand column list inference for inserts.
func (o *Channel) Insert(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) error {
	if o == nil {
		return errors.New("models: no channels provided for insertion")
	}

	var err error
	if !boil.TimestampsAreSkipped(ctx) {
		currTime := time.Now().In(boil.GetLocation())

		if o.CreatedAt.IsZero() {
			o.CreatedAt = currTime
		}
		if o.UpdatedAt.IsZero() {
			o.UpdatedAt = currTime
		}
	}

	nzDefaults := queries.NonZeroDefaultSet(channelColumnsWithDefault, o)

	key := makeCacheKey(columns, nzDefaults)
	channelInsertCacheMut.RLock()
	cache, cached := channelInsertCache[key]
	channelInsertCacheMut.RUnlock()

	if !cached {
		wl, returnColumns := columns.InsertColumnSet(
			channelAllColumns,
			channelColumnsWithDefault,
			channelColumnsWithoutDefault,
			nzDefaults,
		)

		cache.valueMapping, err = queries.BindMapping(channelType, channelMapping, wl)
		if err != nil {
			return err
		}
		cache.retMapping, err = queries.BindMapping(channelType, channelMapping, returnColumns)
		if err != nil {
			return err
		}
		if len(wl) != 0 {
			cache.query = fmt.Sprintf("INSERT INTO \"channels\" (\"%s\") %%sVALUES (%s)%%s", strings.Join(wl, "\",\""), strmangle.Placeholders(dialect.UseIndexPlaceholders, len(wl), 1, 1))
		} else {
			cache.query = "INSERT INTO \"channels\" %sDEFAULT VALUES%s"
		}

		var queryOutput, queryReturning string

		if len(cache.retMapping) != 0 {
			queryReturning = fmt.Sprintf(" RETURNING \"%s\"", strings.Join(returnColumns, "\",\""))
		}

		cache.query = fmt.Sprintf(cache.query, queryOutput, queryReturning)
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.query)
		fmt.Fprintln(boil.DebugWriter, vals)
	}

	if len(cache.retMapping) != 0 {
		err = exec.QueryRowContext(ctx, cache.query, vals...).Scan(queries.PtrsFromMapping(value, cache.retMapping)...)
	} else {
		_, err = exec.ExecContext(ctx, cache.query, vals...)
	}

	if err != nil {
		return errors.Wrap(err, "models: unable to insert into channels")
	}

	if !cached {
		channelInsertCacheMut.Lock()
		channelInsertCache[key] = cache
		channelInsertCacheMut.Unlock()
	}

	return nil
}

// Update uses an executor to update the Channel.
// See boil.Columns.UpdateColumnSet documentation to understand column list inference for updates.
// Update does not automatically update the record in case of default values. Use .Reload() to refresh the records.
func (o *Channel) Update(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) error {
	if !boil.TimestampsAreSkipped(ctx) {
		currTime := time.Now().In(boil.GetLocation())

		o.UpdatedAt = currTime
	}

	var err error
	key := makeCacheKey(columns, nil)
	channelUpdateCacheMut.RLock()
	cache, cached := channelUpdateCache[key]
	channelUpdateCacheMut.RUnlock()

	if !cached {
		wl := columns.UpdateColumnSet(
			channelAllColumns,
			channelPrimaryKeyColumns,
		)

		if !columns.IsWhitelist() {
			wl = strmangle.SetComplement(wl, []string{"created_at"})
		}
		if len(wl) == 0 {
			return errors.New("models: unable to update channels, could not build whitelist")
		}

		cache.query = fmt.Sprintf("UPDATE \"channels\" SET %s WHERE %s",
			strmangle.SetParamNames("\"", "\"", 1, wl),
			strmangle.WhereClause("\"", "\"", len(wl)+1, channelPrimaryKeyColumns),
		)
		cache.valueMapping, err = queries.BindMapping(channelType, channelMapping, append(wl, channelPrimaryKeyColumns...))
		if err != nil {
			return err
		}
	}

	values := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), cache.valueMapping)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.query)
		fmt.Fprintln(boil.DebugWriter, values)
	}

	_, err = exec.ExecContext(ctx, cache.query, values...)
	if err != nil {
		return errors.Wrap(err, "models: unable to update channels row")
	}

	if !cached {
		channelUpdateCacheMut.Lock()
		channelUpdateCache[key] = cache
		channelUpdateCacheMut.Unlock()
	}

	return nil
}

// UpdateAll updates all rows with the specified column values.
func (q channelQuery) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) error {
	queries.SetUpdate(q.Query, cols)

	_, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return errors.Wrap(err, "models: unable to update all for channels")
	}

	return nil
}

// UpdateAll updates all rows with the specified column values, using an executor.
func (o ChannelSlice) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) error {
	ln := int64(len(o))
	if ln == 0 {
		return nil
	}

	if len(cols) == 0 {
		return errors.New("models: update all requires at least one column argument")
	}

	colNames := make([]string, len(cols))
	args := make([]interface{}, len(cols))

	i := 0
	for name, value := range cols {
		colNames[i] = name
		args[i] = value
		i++
	}

	// Append all of the primary key values for each column
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), channelPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf("UPDATE \"channels\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, colNames),
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), len(colNames)+1, channelPrimaryKeyColumns, len(o)))

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}

	_, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to update all in channel slice")
	}

	return nil
}

// Upsert attempts an insert using an executor, and does an update or ignore on conflict.
// See boil.Columns documentation for how to properly use updateColumns and insertColumns.
func (o *Channel) Upsert(ctx context.Context, exec boil.ContextExecutor, updateOnConflict bool, conflictColumns []string, updateColumns, insertColumns boil.Columns) error {
	if o == nil {
		return errors.New("models: no channels provided for upsert")
	}
	if !boil.TimestampsAreSkipped(ctx) {
		currTime := time.Now().In(boil.GetLocation())

		if o.CreatedAt.IsZero() {
			o.CreatedAt = currTime
		}
		o.UpdatedAt = currTime
	}

	nzDefaults := queries.NonZeroDefaultSet(channelColumnsWithDefault, o)

	// Build cache key in-line uglily - mysql vs psql problems
	buf := strmangle.GetBuffer()
	if updateOnConflict {
		buf.WriteByte('t')
	} else {
		buf.WriteByte('f')
	}
	buf.WriteByte('.')
	for _, c := range conflictColumns {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	buf.WriteString(strconv.Itoa(updateColumns.Kind))
	for _, c := range updateColumns.Cols {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	buf.WriteString(strconv.Itoa(insertColumns.Kind))
	for _, c := range insertColumns.Cols {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	for _, c := range nzDefaults {
		buf.WriteString(c)
	}
	key := buf.String()
	strmangle.PutBuffer(buf)

	channelUpsertCacheMut.RLock()
	cache, cached := channelUpsertCache[key]
	channelUpsertCacheMut.RUnlock()

	var err error

	if !cached {
		insert, ret := insertColumns.InsertColumnSet(
			channelAllColumns,
			channelColumnsWithDefault,
			channelColumnsWithoutDefault,
			nzDefaults,
		)
		update := updateColumns.UpdateColumnSet(
			channelAllColumns,
			channelPrimaryKeyColumns,
		)

		if updateOnConflict && len(update) == 0 {
			return errors.New("models: unable to upsert channels, could not build update column list")
		}

		conflict := conflictColumns
		if len(conflict) == 0 {
			conflict = make([]string, len(channelPrimaryKeyColumns))
			copy(conflict, channelPrimaryKeyColumns)
		}
		cache.query = buildUpsertQueryPostgres(dialect, "\"channels\"", updateOnConflict, ret, update, conflict, insert)

		cache.valueMapping, err = queries.BindMapping(channelType, channelMapping, insert)
		if err != nil {
			return err
		}
		if len(ret) != 0 {
			cache.retMapping, err = queries.BindMapping(channelType, channelMapping, ret)
			if err != nil {
				return err
			}
		}
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)
	var returns []interface{}
	if len(cache.retMapping) != 0 {
		returns = queries.PtrsFromMapping(value, cache.retMapping)
	}

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.query)
		fmt.Fprintln(boil.DebugWriter, vals)
	}

	if len(cache.retMapping) != 0 {
		err = exec.QueryRowContext(ctx, cache.query, vals...).Scan(returns...)
		if err == sql.ErrNoRows {
			err = nil // Postgres doesn't return anything when there's no update
		}
	} else {
		_, err = exec.ExecContext(ctx, cache.query, vals...)
	}
	if err != nil {
		return errors.Wrap(err, "models: unable to upsert channels")
	}

	if !cached {
		channelUpsertCacheMut.Lock()
		channelUpsertCache[key] = cache
		channelUpsertCacheMut.Unlock()
	}

	return nil
}

// Delete deletes a single Channel record with an executor.
// Delete will match against the primary key column to find the record to delete.
func (o *Channel) Delete(ctx context.Context, exec boil.ContextExecutor) error {
	if o == nil {
		return errors.New("models: no Channel provided for delete")
	}

	args := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), channelPrimaryKeyMapping)
	sql := "DELETE FROM \"channels\" WHERE \"id\"=$1"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}

	_, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete from channels")
	}

	return nil
}

// DeleteAll deletes all matching rows.
func (q channelQuery) DeleteAll(ctx context.Context, exec boil.ContextExecutor) error {
	if q.Query == nil {
		return errors.New("models: no channelQuery provided for delete all")
	}

	queries.SetDelete(q.Query)

	_, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete all from channels")
	}

	return nil
}

// DeleteAll deletes all rows in the slice, using an executor.
func (o ChannelSlice) DeleteAll(ctx context.Context, exec boil.ContextExecutor) error {
	if len(o) == 0 {
		return nil
	}

	var args []interface{}
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), channelPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "DELETE FROM \"channels\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, channelPrimaryKeyColumns, len(o))

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args)
	}

	_, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete all from channel slice")
	}

	return nil
}

// Reload refetches the object from the database
// using the primary keys with an executor.
func (o *Channel) Reload(ctx context.Context, exec boil.ContextExecutor) error {
	ret, err := FindChannel(ctx, exec, o.ID)
	if err != nil {
		return err
	}

	*o = *ret
	return nil
}

// ReloadAll refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *ChannelSlice) ReloadAll(ctx context.Context, exec boil.ContextExecutor) error {
	if o == nil || len(*o) == 0 {
		return nil
	}

	slice := ChannelSlice{}
	var args []interface{}
	for _, obj := range *o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), channelPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "SELECT \"channels\".* FROM \"channels\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, channelPrimaryKeyColumns, len(*o))

	q := queries.Raw(sql, args...)

	err := q.Bind(ctx, exec, &slice)
	if err != nil {
		return errors.Wrap(err, "models: unable to reload all in ChannelSlice")
	}

	*o = slice

	return nil
}

// ChannelExists checks if the Channel row exists.
func ChannelExists(ctx context.Context, exec boil.ContextExecutor, iD int64) (bool, error) {
	var exists bool
	sql := "select exists(select 1 from \"channels\" where \"id\"=$1 limit 1)"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, iD)
	}

	row := exec.QueryRowContext(ctx, sql, iD)

	err := row.Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "models: unable to check if channels exists")
	}

	return exists, nil
}
