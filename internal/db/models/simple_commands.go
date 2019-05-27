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
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"github.com/volatiletech/sqlboiler/queries/qmhelper"
	"github.com/volatiletech/sqlboiler/strmangle"
)

// SimpleCommand is an object representing the database table.
type SimpleCommand struct {
	ID          int64     `boil:"id" json:"id" toml:"id" yaml:"id"`
	CreatedAt   time.Time `boil:"created_at" json:"created_at" toml:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time `boil:"updated_at" json:"updated_at" toml:"updated_at" yaml:"updated_at"`
	ChannelID   int64     `boil:"channel_id" json:"channel_id" toml:"channel_id" yaml:"channel_id"`
	Name        string    `boil:"name" json:"name" toml:"name" yaml:"name"`
	Message     string    `boil:"message" json:"message" toml:"message" yaml:"message"`
	Editor      string    `boil:"editor" json:"editor" toml:"editor" yaml:"editor"`
	AccessLevel string    `boil:"access_level" json:"access_level" toml:"access_level" yaml:"access_level"`
	Count       int64     `boil:"count" json:"count" toml:"count" yaml:"count"`

	R *simpleCommandR `boil:"-" json:"-" toml:"-" yaml:"-"`
	L simpleCommandL  `boil:"-" json:"-" toml:"-" yaml:"-"`
}

var SimpleCommandColumns = struct {
	ID          string
	CreatedAt   string
	UpdatedAt   string
	ChannelID   string
	Name        string
	Message     string
	Editor      string
	AccessLevel string
	Count       string
}{
	ID:          "id",
	CreatedAt:   "created_at",
	UpdatedAt:   "updated_at",
	ChannelID:   "channel_id",
	Name:        "name",
	Message:     "message",
	Editor:      "editor",
	AccessLevel: "access_level",
	Count:       "count",
}

// Generated where

var SimpleCommandWhere = struct {
	ID          whereHelperint64
	CreatedAt   whereHelpertime_Time
	UpdatedAt   whereHelpertime_Time
	ChannelID   whereHelperint64
	Name        whereHelperstring
	Message     whereHelperstring
	Editor      whereHelperstring
	AccessLevel whereHelperstring
	Count       whereHelperint64
}{
	ID:          whereHelperint64{field: "\"simple_commands\".\"id\""},
	CreatedAt:   whereHelpertime_Time{field: "\"simple_commands\".\"created_at\""},
	UpdatedAt:   whereHelpertime_Time{field: "\"simple_commands\".\"updated_at\""},
	ChannelID:   whereHelperint64{field: "\"simple_commands\".\"channel_id\""},
	Name:        whereHelperstring{field: "\"simple_commands\".\"name\""},
	Message:     whereHelperstring{field: "\"simple_commands\".\"message\""},
	Editor:      whereHelperstring{field: "\"simple_commands\".\"editor\""},
	AccessLevel: whereHelperstring{field: "\"simple_commands\".\"access_level\""},
	Count:       whereHelperint64{field: "\"simple_commands\".\"count\""},
}

// SimpleCommandRels is where relationship names are stored.
var SimpleCommandRels = struct {
	Channel string
}{
	Channel: "Channel",
}

// simpleCommandR is where relationships are stored.
type simpleCommandR struct {
	Channel *Channel
}

// NewStruct creates a new relationship struct
func (*simpleCommandR) NewStruct() *simpleCommandR {
	return &simpleCommandR{}
}

// simpleCommandL is where Load methods for each relationship are stored.
type simpleCommandL struct{}

var (
	simpleCommandAllColumns            = []string{"id", "created_at", "updated_at", "channel_id", "name", "message", "editor", "access_level", "count"}
	simpleCommandColumnsWithoutDefault = []string{"channel_id", "name", "message", "editor", "access_level", "count"}
	simpleCommandColumnsWithDefault    = []string{"id", "created_at", "updated_at"}
	simpleCommandPrimaryKeyColumns     = []string{"id"}
)

type (
	// SimpleCommandSlice is an alias for a slice of pointers to SimpleCommand.
	// This should generally be used opposed to []SimpleCommand.
	SimpleCommandSlice []*SimpleCommand

	simpleCommandQuery struct {
		*queries.Query
	}
)

// Cache for insert, update and upsert
var (
	simpleCommandType                 = reflect.TypeOf(&SimpleCommand{})
	simpleCommandMapping              = queries.MakeStructMapping(simpleCommandType)
	simpleCommandPrimaryKeyMapping, _ = queries.BindMapping(simpleCommandType, simpleCommandMapping, simpleCommandPrimaryKeyColumns)
	simpleCommandInsertCacheMut       sync.RWMutex
	simpleCommandInsertCache          = make(map[string]insertCache)
	simpleCommandUpdateCacheMut       sync.RWMutex
	simpleCommandUpdateCache          = make(map[string]updateCache)
	simpleCommandUpsertCacheMut       sync.RWMutex
	simpleCommandUpsertCache          = make(map[string]insertCache)
)

var (
	// Force time package dependency for automated UpdatedAt/CreatedAt.
	_ = time.Second
	// Force qmhelper dependency for where clause generation (which doesn't
	// always happen)
	_ = qmhelper.Where
)

// One returns a single simpleCommand record from the query.
func (q simpleCommandQuery) One(ctx context.Context, exec boil.ContextExecutor) (*SimpleCommand, error) {
	o := &SimpleCommand{}

	queries.SetLimit(q.Query, 1)

	err := q.Bind(ctx, exec, o)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: failed to execute a one query for simple_commands")
	}

	return o, nil
}

// All returns all SimpleCommand records from the query.
func (q simpleCommandQuery) All(ctx context.Context, exec boil.ContextExecutor) (SimpleCommandSlice, error) {
	var o []*SimpleCommand

	err := q.Bind(ctx, exec, &o)
	if err != nil {
		return nil, errors.Wrap(err, "models: failed to assign all query results to SimpleCommand slice")
	}

	return o, nil
}

// Count returns the count of all SimpleCommand records in the query.
func (q simpleCommandQuery) Count(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to count simple_commands rows")
	}

	return count, nil
}

// Exists checks if the row exists in the table.
func (q simpleCommandQuery) Exists(ctx context.Context, exec boil.ContextExecutor) (bool, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)
	queries.SetLimit(q.Query, 1)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "models: failed to check if simple_commands exists")
	}

	return count > 0, nil
}

// Channel pointed to by the foreign key.
func (o *SimpleCommand) Channel(mods ...qm.QueryMod) channelQuery {
	queryMods := []qm.QueryMod{
		qm.Where("id=?", o.ChannelID),
	}

	queryMods = append(queryMods, mods...)

	query := Channels(queryMods...)
	queries.SetFrom(query.Query, "\"channels\"")

	return query
}

// LoadChannel allows an eager lookup of values, cached into the
// loaded structs of the objects. This is for an N-1 relationship.
func (simpleCommandL) LoadChannel(ctx context.Context, e boil.ContextExecutor, singular bool, maybeSimpleCommand interface{}, mods queries.Applicator) error {
	var slice []*SimpleCommand
	var object *SimpleCommand

	if singular {
		object = maybeSimpleCommand.(*SimpleCommand)
	} else {
		slice = *maybeSimpleCommand.(*[]*SimpleCommand)
	}

	args := make([]interface{}, 0, 1)
	if singular {
		if object.R == nil {
			object.R = &simpleCommandR{}
		}
		args = append(args, object.ChannelID)

	} else {
	Outer:
		for _, obj := range slice {
			if obj.R == nil {
				obj.R = &simpleCommandR{}
			}

			for _, a := range args {
				if a == obj.ChannelID {
					continue Outer
				}
			}

			args = append(args, obj.ChannelID)

		}
	}

	if len(args) == 0 {
		return nil
	}

	query := NewQuery(qm.From(`channels`), qm.WhereIn(`id in ?`, args...))
	if mods != nil {
		mods.Apply(query)
	}

	results, err := query.QueryContext(ctx, e)
	if err != nil {
		return errors.Wrap(err, "failed to eager load Channel")
	}

	var resultSlice []*Channel
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice Channel")
	}

	if err = results.Close(); err != nil {
		return errors.Wrap(err, "failed to close results of eager load for channels")
	}
	if err = results.Err(); err != nil {
		return errors.Wrap(err, "error occurred during iteration of eager loaded relations for channels")
	}

	if len(resultSlice) == 0 {
		return nil
	}

	if singular {
		foreign := resultSlice[0]
		object.R.Channel = foreign
		if foreign.R == nil {
			foreign.R = &channelR{}
		}
		foreign.R.SimpleCommands = append(foreign.R.SimpleCommands, object)
		return nil
	}

	for _, local := range slice {
		for _, foreign := range resultSlice {
			if local.ChannelID == foreign.ID {
				local.R.Channel = foreign
				if foreign.R == nil {
					foreign.R = &channelR{}
				}
				foreign.R.SimpleCommands = append(foreign.R.SimpleCommands, local)
				break
			}
		}
	}

	return nil
}

// SetChannel of the simpleCommand to the related item.
// Sets o.R.Channel to related.
// Adds o to related.R.SimpleCommands.
func (o *SimpleCommand) SetChannel(ctx context.Context, exec boil.ContextExecutor, insert bool, related *Channel) error {
	var err error
	if insert {
		if err = related.Insert(ctx, exec, boil.Infer()); err != nil {
			return errors.Wrap(err, "failed to insert into foreign table")
		}
	}

	updateQuery := fmt.Sprintf(
		"UPDATE \"simple_commands\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, []string{"channel_id"}),
		strmangle.WhereClause("\"", "\"", 2, simpleCommandPrimaryKeyColumns),
	)
	values := []interface{}{related.ID, o.ID}

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, updateQuery)
		fmt.Fprintln(boil.DebugWriter, values)
	}

	if _, err = exec.ExecContext(ctx, updateQuery, values...); err != nil {
		return errors.Wrap(err, "failed to update local table")
	}

	o.ChannelID = related.ID
	if o.R == nil {
		o.R = &simpleCommandR{
			Channel: related,
		}
	} else {
		o.R.Channel = related
	}

	if related.R == nil {
		related.R = &channelR{
			SimpleCommands: SimpleCommandSlice{o},
		}
	} else {
		related.R.SimpleCommands = append(related.R.SimpleCommands, o)
	}

	return nil
}

// SimpleCommands retrieves all the records using an executor.
func SimpleCommands(mods ...qm.QueryMod) simpleCommandQuery {
	mods = append(mods, qm.From("\"simple_commands\""))
	return simpleCommandQuery{NewQuery(mods...)}
}

// FindSimpleCommand retrieves a single record by ID with an executor.
// If selectCols is empty Find will return all columns.
func FindSimpleCommand(ctx context.Context, exec boil.ContextExecutor, iD int64, selectCols ...string) (*SimpleCommand, error) {
	simpleCommandObj := &SimpleCommand{}

	sel := "*"
	if len(selectCols) > 0 {
		sel = strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, selectCols), ",")
	}
	query := fmt.Sprintf(
		"select %s from \"simple_commands\" where \"id\"=$1", sel,
	)

	q := queries.Raw(query, iD)

	err := q.Bind(ctx, exec, simpleCommandObj)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: unable to select from simple_commands")
	}

	return simpleCommandObj, nil
}

// Insert a single record using an executor.
// See boil.Columns.InsertColumnSet documentation to understand column list inference for inserts.
func (o *SimpleCommand) Insert(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) error {
	if o == nil {
		return errors.New("models: no simple_commands provided for insertion")
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

	nzDefaults := queries.NonZeroDefaultSet(simpleCommandColumnsWithDefault, o)

	key := makeCacheKey(columns, nzDefaults)
	simpleCommandInsertCacheMut.RLock()
	cache, cached := simpleCommandInsertCache[key]
	simpleCommandInsertCacheMut.RUnlock()

	if !cached {
		wl, returnColumns := columns.InsertColumnSet(
			simpleCommandAllColumns,
			simpleCommandColumnsWithDefault,
			simpleCommandColumnsWithoutDefault,
			nzDefaults,
		)

		cache.valueMapping, err = queries.BindMapping(simpleCommandType, simpleCommandMapping, wl)
		if err != nil {
			return err
		}
		cache.retMapping, err = queries.BindMapping(simpleCommandType, simpleCommandMapping, returnColumns)
		if err != nil {
			return err
		}
		if len(wl) != 0 {
			cache.query = fmt.Sprintf("INSERT INTO \"simple_commands\" (\"%s\") %%sVALUES (%s)%%s", strings.Join(wl, "\",\""), strmangle.Placeholders(dialect.UseIndexPlaceholders, len(wl), 1, 1))
		} else {
			cache.query = "INSERT INTO \"simple_commands\" %sDEFAULT VALUES%s"
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
		return errors.Wrap(err, "models: unable to insert into simple_commands")
	}

	if !cached {
		simpleCommandInsertCacheMut.Lock()
		simpleCommandInsertCache[key] = cache
		simpleCommandInsertCacheMut.Unlock()
	}

	return nil
}

// Update uses an executor to update the SimpleCommand.
// See boil.Columns.UpdateColumnSet documentation to understand column list inference for updates.
// Update does not automatically update the record in case of default values. Use .Reload() to refresh the records.
func (o *SimpleCommand) Update(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) error {
	if !boil.TimestampsAreSkipped(ctx) {
		currTime := time.Now().In(boil.GetLocation())

		o.UpdatedAt = currTime
	}

	var err error
	key := makeCacheKey(columns, nil)
	simpleCommandUpdateCacheMut.RLock()
	cache, cached := simpleCommandUpdateCache[key]
	simpleCommandUpdateCacheMut.RUnlock()

	if !cached {
		wl := columns.UpdateColumnSet(
			simpleCommandAllColumns,
			simpleCommandPrimaryKeyColumns,
		)

		if !columns.IsWhitelist() {
			wl = strmangle.SetComplement(wl, []string{"created_at"})
		}
		if len(wl) == 0 {
			return errors.New("models: unable to update simple_commands, could not build whitelist")
		}

		cache.query = fmt.Sprintf("UPDATE \"simple_commands\" SET %s WHERE %s",
			strmangle.SetParamNames("\"", "\"", 1, wl),
			strmangle.WhereClause("\"", "\"", len(wl)+1, simpleCommandPrimaryKeyColumns),
		)
		cache.valueMapping, err = queries.BindMapping(simpleCommandType, simpleCommandMapping, append(wl, simpleCommandPrimaryKeyColumns...))
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
		return errors.Wrap(err, "models: unable to update simple_commands row")
	}

	if !cached {
		simpleCommandUpdateCacheMut.Lock()
		simpleCommandUpdateCache[key] = cache
		simpleCommandUpdateCacheMut.Unlock()
	}

	return nil
}

// UpdateAll updates all rows with the specified column values.
func (q simpleCommandQuery) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) error {
	queries.SetUpdate(q.Query, cols)

	_, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return errors.Wrap(err, "models: unable to update all for simple_commands")
	}

	return nil
}

// UpdateAll updates all rows with the specified column values, using an executor.
func (o SimpleCommandSlice) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) error {
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
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), simpleCommandPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf("UPDATE \"simple_commands\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, colNames),
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), len(colNames)+1, simpleCommandPrimaryKeyColumns, len(o)))

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}

	_, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to update all in simpleCommand slice")
	}

	return nil
}

// Upsert attempts an insert using an executor, and does an update or ignore on conflict.
// See boil.Columns documentation for how to properly use updateColumns and insertColumns.
func (o *SimpleCommand) Upsert(ctx context.Context, exec boil.ContextExecutor, updateOnConflict bool, conflictColumns []string, updateColumns, insertColumns boil.Columns) error {
	if o == nil {
		return errors.New("models: no simple_commands provided for upsert")
	}
	if !boil.TimestampsAreSkipped(ctx) {
		currTime := time.Now().In(boil.GetLocation())

		if o.CreatedAt.IsZero() {
			o.CreatedAt = currTime
		}
		o.UpdatedAt = currTime
	}

	nzDefaults := queries.NonZeroDefaultSet(simpleCommandColumnsWithDefault, o)

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

	simpleCommandUpsertCacheMut.RLock()
	cache, cached := simpleCommandUpsertCache[key]
	simpleCommandUpsertCacheMut.RUnlock()

	var err error

	if !cached {
		insert, ret := insertColumns.InsertColumnSet(
			simpleCommandAllColumns,
			simpleCommandColumnsWithDefault,
			simpleCommandColumnsWithoutDefault,
			nzDefaults,
		)
		update := updateColumns.UpdateColumnSet(
			simpleCommandAllColumns,
			simpleCommandPrimaryKeyColumns,
		)

		if updateOnConflict && len(update) == 0 {
			return errors.New("models: unable to upsert simple_commands, could not build update column list")
		}

		conflict := conflictColumns
		if len(conflict) == 0 {
			conflict = make([]string, len(simpleCommandPrimaryKeyColumns))
			copy(conflict, simpleCommandPrimaryKeyColumns)
		}
		cache.query = buildUpsertQueryPostgres(dialect, "\"simple_commands\"", updateOnConflict, ret, update, conflict, insert)

		cache.valueMapping, err = queries.BindMapping(simpleCommandType, simpleCommandMapping, insert)
		if err != nil {
			return err
		}
		if len(ret) != 0 {
			cache.retMapping, err = queries.BindMapping(simpleCommandType, simpleCommandMapping, ret)
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
		return errors.Wrap(err, "models: unable to upsert simple_commands")
	}

	if !cached {
		simpleCommandUpsertCacheMut.Lock()
		simpleCommandUpsertCache[key] = cache
		simpleCommandUpsertCacheMut.Unlock()
	}

	return nil
}

// Delete deletes a single SimpleCommand record with an executor.
// Delete will match against the primary key column to find the record to delete.
func (o *SimpleCommand) Delete(ctx context.Context, exec boil.ContextExecutor) error {
	if o == nil {
		return errors.New("models: no SimpleCommand provided for delete")
	}

	args := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), simpleCommandPrimaryKeyMapping)
	sql := "DELETE FROM \"simple_commands\" WHERE \"id\"=$1"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}

	_, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete from simple_commands")
	}

	return nil
}

// DeleteAll deletes all matching rows.
func (q simpleCommandQuery) DeleteAll(ctx context.Context, exec boil.ContextExecutor) error {
	if q.Query == nil {
		return errors.New("models: no simpleCommandQuery provided for delete all")
	}

	queries.SetDelete(q.Query)

	_, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete all from simple_commands")
	}

	return nil
}

// DeleteAll deletes all rows in the slice, using an executor.
func (o SimpleCommandSlice) DeleteAll(ctx context.Context, exec boil.ContextExecutor) error {
	if len(o) == 0 {
		return nil
	}

	var args []interface{}
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), simpleCommandPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "DELETE FROM \"simple_commands\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, simpleCommandPrimaryKeyColumns, len(o))

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args)
	}

	_, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete all from simpleCommand slice")
	}

	return nil
}

// Reload refetches the object from the database
// using the primary keys with an executor.
func (o *SimpleCommand) Reload(ctx context.Context, exec boil.ContextExecutor) error {
	ret, err := FindSimpleCommand(ctx, exec, o.ID)
	if err != nil {
		return err
	}

	*o = *ret
	return nil
}

// ReloadAll refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *SimpleCommandSlice) ReloadAll(ctx context.Context, exec boil.ContextExecutor) error {
	if o == nil || len(*o) == 0 {
		return nil
	}

	slice := SimpleCommandSlice{}
	var args []interface{}
	for _, obj := range *o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), simpleCommandPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "SELECT \"simple_commands\".* FROM \"simple_commands\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, simpleCommandPrimaryKeyColumns, len(*o))

	q := queries.Raw(sql, args...)

	err := q.Bind(ctx, exec, &slice)
	if err != nil {
		return errors.Wrap(err, "models: unable to reload all in SimpleCommandSlice")
	}

	*o = slice

	return nil
}

// SimpleCommandExists checks if the SimpleCommand row exists.
func SimpleCommandExists(ctx context.Context, exec boil.ContextExecutor, iD int64) (bool, error) {
	var exists bool
	sql := "select exists(select 1 from \"simple_commands\" where \"id\"=$1 limit 1)"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, iD)
	}

	row := exec.QueryRowContext(ctx, sql, iD)

	err := row.Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "models: unable to check if simple_commands exists")
	}

	return exists, nil
}
