// Code generated by SQLBoiler 4.18.0 (https://github.com/volatiletech/sqlboiler). DO NOT EDIT.
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

	"github.com/friendsofgo/errors"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"github.com/volatiletech/sqlboiler/v4/queries/qmhelper"
	"github.com/volatiletech/strmangle"
)

// ModeratedChannel is an object representing the database table.
type ModeratedChannel struct {
	ID               int64     `boil:"id" json:"id" toml:"id" yaml:"id"`
	CreatedAt        time.Time `boil:"created_at" json:"created_at" toml:"created_at" yaml:"created_at"`
	UpdatedAt        time.Time `boil:"updated_at" json:"updated_at" toml:"updated_at" yaml:"updated_at"`
	BotName          string    `boil:"bot_name" json:"bot_name" toml:"bot_name" yaml:"bot_name"`
	BroadcasterID    int64     `boil:"broadcaster_id" json:"broadcaster_id" toml:"broadcaster_id" yaml:"broadcaster_id"`
	BroadcasterLogin string    `boil:"broadcaster_login" json:"broadcaster_login" toml:"broadcaster_login" yaml:"broadcaster_login"`
	BroadcasterName  string    `boil:"broadcaster_name" json:"broadcaster_name" toml:"broadcaster_name" yaml:"broadcaster_name"`

	R *moderatedChannelR `boil:"-" json:"-" toml:"-" yaml:"-"`
	L moderatedChannelL  `boil:"-" json:"-" toml:"-" yaml:"-"`
}

var ModeratedChannelColumns = struct {
	ID               string
	CreatedAt        string
	UpdatedAt        string
	BotName          string
	BroadcasterID    string
	BroadcasterLogin string
	BroadcasterName  string
}{
	ID:               "id",
	CreatedAt:        "created_at",
	UpdatedAt:        "updated_at",
	BotName:          "bot_name",
	BroadcasterID:    "broadcaster_id",
	BroadcasterLogin: "broadcaster_login",
	BroadcasterName:  "broadcaster_name",
}

var ModeratedChannelTableColumns = struct {
	ID               string
	CreatedAt        string
	UpdatedAt        string
	BotName          string
	BroadcasterID    string
	BroadcasterLogin string
	BroadcasterName  string
}{
	ID:               "moderated_channels.id",
	CreatedAt:        "moderated_channels.created_at",
	UpdatedAt:        "moderated_channels.updated_at",
	BotName:          "moderated_channels.bot_name",
	BroadcasterID:    "moderated_channels.broadcaster_id",
	BroadcasterLogin: "moderated_channels.broadcaster_login",
	BroadcasterName:  "moderated_channels.broadcaster_name",
}

// Generated where

var ModeratedChannelWhere = struct {
	ID               whereHelperint64
	CreatedAt        whereHelpertime_Time
	UpdatedAt        whereHelpertime_Time
	BotName          whereHelperstring
	BroadcasterID    whereHelperint64
	BroadcasterLogin whereHelperstring
	BroadcasterName  whereHelperstring
}{
	ID:               whereHelperint64{field: "\"moderated_channels\".\"id\""},
	CreatedAt:        whereHelpertime_Time{field: "\"moderated_channels\".\"created_at\""},
	UpdatedAt:        whereHelpertime_Time{field: "\"moderated_channels\".\"updated_at\""},
	BotName:          whereHelperstring{field: "\"moderated_channels\".\"bot_name\""},
	BroadcasterID:    whereHelperint64{field: "\"moderated_channels\".\"broadcaster_id\""},
	BroadcasterLogin: whereHelperstring{field: "\"moderated_channels\".\"broadcaster_login\""},
	BroadcasterName:  whereHelperstring{field: "\"moderated_channels\".\"broadcaster_name\""},
}

// ModeratedChannelRels is where relationship names are stored.
var ModeratedChannelRels = struct {
}{}

// moderatedChannelR is where relationships are stored.
type moderatedChannelR struct {
}

// NewStruct creates a new relationship struct
func (*moderatedChannelR) NewStruct() *moderatedChannelR {
	return &moderatedChannelR{}
}

// moderatedChannelL is where Load methods for each relationship are stored.
type moderatedChannelL struct{}

var (
	moderatedChannelAllColumns            = []string{"id", "created_at", "updated_at", "bot_name", "broadcaster_id", "broadcaster_login", "broadcaster_name"}
	moderatedChannelColumnsWithoutDefault = []string{"bot_name", "broadcaster_id", "broadcaster_login", "broadcaster_name"}
	moderatedChannelColumnsWithDefault    = []string{"id", "created_at", "updated_at"}
	moderatedChannelPrimaryKeyColumns     = []string{"id"}
	moderatedChannelGeneratedColumns      = []string{}
)

type (
	// ModeratedChannelSlice is an alias for a slice of pointers to ModeratedChannel.
	// This should almost always be used instead of []ModeratedChannel.
	ModeratedChannelSlice []*ModeratedChannel

	moderatedChannelQuery struct {
		*queries.Query
	}
)

// Cache for insert, update and upsert
var (
	moderatedChannelType                 = reflect.TypeOf(&ModeratedChannel{})
	moderatedChannelMapping              = queries.MakeStructMapping(moderatedChannelType)
	moderatedChannelPrimaryKeyMapping, _ = queries.BindMapping(moderatedChannelType, moderatedChannelMapping, moderatedChannelPrimaryKeyColumns)
	moderatedChannelInsertCacheMut       sync.RWMutex
	moderatedChannelInsertCache          = make(map[string]insertCache)
	moderatedChannelUpdateCacheMut       sync.RWMutex
	moderatedChannelUpdateCache          = make(map[string]updateCache)
	moderatedChannelUpsertCacheMut       sync.RWMutex
	moderatedChannelUpsertCache          = make(map[string]insertCache)
)

var (
	// Force time package dependency for automated UpdatedAt/CreatedAt.
	_ = time.Second
	// Force qmhelper dependency for where clause generation (which doesn't
	// always happen)
	_ = qmhelper.Where
)

// One returns a single moderatedChannel record from the query.
func (q moderatedChannelQuery) One(ctx context.Context, exec boil.ContextExecutor) (*ModeratedChannel, error) {
	o := &ModeratedChannel{}

	queries.SetLimit(q.Query, 1)

	err := q.Bind(ctx, exec, o)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: failed to execute a one query for moderated_channels")
	}

	return o, nil
}

// All returns all ModeratedChannel records from the query.
func (q moderatedChannelQuery) All(ctx context.Context, exec boil.ContextExecutor) (ModeratedChannelSlice, error) {
	var o []*ModeratedChannel

	err := q.Bind(ctx, exec, &o)
	if err != nil {
		return nil, errors.Wrap(err, "models: failed to assign all query results to ModeratedChannel slice")
	}

	return o, nil
}

// Count returns the count of all ModeratedChannel records in the query.
func (q moderatedChannelQuery) Count(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to count moderated_channels rows")
	}

	return count, nil
}

// Exists checks if the row exists in the table.
func (q moderatedChannelQuery) Exists(ctx context.Context, exec boil.ContextExecutor) (bool, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)
	queries.SetLimit(q.Query, 1)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "models: failed to check if moderated_channels exists")
	}

	return count > 0, nil
}

// ModeratedChannels retrieves all the records using an executor.
func ModeratedChannels(mods ...qm.QueryMod) moderatedChannelQuery {
	mods = append(mods, qm.From("\"moderated_channels\""))
	q := NewQuery(mods...)
	if len(queries.GetSelect(q)) == 0 {
		queries.SetSelect(q, []string{"\"moderated_channels\".*"})
	}

	return moderatedChannelQuery{q}
}

// FindModeratedChannel retrieves a single record by ID with an executor.
// If selectCols is empty Find will return all columns.
func FindModeratedChannel(ctx context.Context, exec boil.ContextExecutor, iD int64, selectCols ...string) (*ModeratedChannel, error) {
	moderatedChannelObj := &ModeratedChannel{}

	sel := "*"
	if len(selectCols) > 0 {
		sel = strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, selectCols), ",")
	}
	query := fmt.Sprintf(
		"select %s from \"moderated_channels\" where \"id\"=$1", sel,
	)

	q := queries.Raw(query, iD)

	err := q.Bind(ctx, exec, moderatedChannelObj)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: unable to select from moderated_channels")
	}

	return moderatedChannelObj, nil
}

// Insert a single record using an executor.
// See boil.Columns.InsertColumnSet documentation to understand column list inference for inserts.
func (o *ModeratedChannel) Insert(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) error {
	if o == nil {
		return errors.New("models: no moderated_channels provided for insertion")
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

	nzDefaults := queries.NonZeroDefaultSet(moderatedChannelColumnsWithDefault, o)

	key := makeCacheKey(columns, nzDefaults)
	moderatedChannelInsertCacheMut.RLock()
	cache, cached := moderatedChannelInsertCache[key]
	moderatedChannelInsertCacheMut.RUnlock()

	if !cached {
		wl, returnColumns := columns.InsertColumnSet(
			moderatedChannelAllColumns,
			moderatedChannelColumnsWithDefault,
			moderatedChannelColumnsWithoutDefault,
			nzDefaults,
		)

		cache.valueMapping, err = queries.BindMapping(moderatedChannelType, moderatedChannelMapping, wl)
		if err != nil {
			return err
		}
		cache.retMapping, err = queries.BindMapping(moderatedChannelType, moderatedChannelMapping, returnColumns)
		if err != nil {
			return err
		}
		if len(wl) != 0 {
			cache.query = fmt.Sprintf("INSERT INTO \"moderated_channels\" (\"%s\") %%sVALUES (%s)%%s", strings.Join(wl, "\",\""), strmangle.Placeholders(dialect.UseIndexPlaceholders, len(wl), 1, 1))
		} else {
			cache.query = "INSERT INTO \"moderated_channels\" %sDEFAULT VALUES%s"
		}

		var queryOutput, queryReturning string

		if len(cache.retMapping) != 0 {
			queryReturning = fmt.Sprintf(" RETURNING \"%s\"", strings.Join(returnColumns, "\",\""))
		}

		cache.query = fmt.Sprintf(cache.query, queryOutput, queryReturning)
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, cache.query)
		fmt.Fprintln(writer, vals)
	}

	if len(cache.retMapping) != 0 {
		err = exec.QueryRowContext(ctx, cache.query, vals...).Scan(queries.PtrsFromMapping(value, cache.retMapping)...)
	} else {
		_, err = exec.ExecContext(ctx, cache.query, vals...)
	}

	if err != nil {
		return errors.Wrap(err, "models: unable to insert into moderated_channels")
	}

	if !cached {
		moderatedChannelInsertCacheMut.Lock()
		moderatedChannelInsertCache[key] = cache
		moderatedChannelInsertCacheMut.Unlock()
	}

	return nil
}

// Update uses an executor to update the ModeratedChannel.
// See boil.Columns.UpdateColumnSet documentation to understand column list inference for updates.
// Update does not automatically update the record in case of default values. Use .Reload() to refresh the records.
func (o *ModeratedChannel) Update(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) error {
	if !boil.TimestampsAreSkipped(ctx) {
		currTime := time.Now().In(boil.GetLocation())

		o.UpdatedAt = currTime
	}

	var err error
	key := makeCacheKey(columns, nil)
	moderatedChannelUpdateCacheMut.RLock()
	cache, cached := moderatedChannelUpdateCache[key]
	moderatedChannelUpdateCacheMut.RUnlock()

	if !cached {
		wl := columns.UpdateColumnSet(
			moderatedChannelAllColumns,
			moderatedChannelPrimaryKeyColumns,
		)

		if !columns.IsWhitelist() {
			wl = strmangle.SetComplement(wl, []string{"created_at"})
		}
		if len(wl) == 0 {
			return errors.New("models: unable to update moderated_channels, could not build whitelist")
		}

		cache.query = fmt.Sprintf("UPDATE \"moderated_channels\" SET %s WHERE %s",
			strmangle.SetParamNames("\"", "\"", 1, wl),
			strmangle.WhereClause("\"", "\"", len(wl)+1, moderatedChannelPrimaryKeyColumns),
		)
		cache.valueMapping, err = queries.BindMapping(moderatedChannelType, moderatedChannelMapping, append(wl, moderatedChannelPrimaryKeyColumns...))
		if err != nil {
			return err
		}
	}

	values := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), cache.valueMapping)

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, cache.query)
		fmt.Fprintln(writer, values)
	}
	_, err = exec.ExecContext(ctx, cache.query, values...)
	if err != nil {
		return errors.Wrap(err, "models: unable to update moderated_channels row")
	}

	if !cached {
		moderatedChannelUpdateCacheMut.Lock()
		moderatedChannelUpdateCache[key] = cache
		moderatedChannelUpdateCacheMut.Unlock()
	}

	return nil
}

// UpdateAll updates all rows with the specified column values.
func (q moderatedChannelQuery) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) error {
	queries.SetUpdate(q.Query, cols)

	_, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return errors.Wrap(err, "models: unable to update all for moderated_channels")
	}

	return nil
}

// UpdateAll updates all rows with the specified column values, using an executor.
func (o ModeratedChannelSlice) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) error {
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
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), moderatedChannelPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf("UPDATE \"moderated_channels\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, colNames),
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), len(colNames)+1, moderatedChannelPrimaryKeyColumns, len(o)))

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, args...)
	}
	_, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to update all in moderatedChannel slice")
	}

	return nil
}

// Upsert attempts an insert using an executor, and does an update or ignore on conflict.
// See boil.Columns documentation for how to properly use updateColumns and insertColumns.
func (o *ModeratedChannel) Upsert(ctx context.Context, exec boil.ContextExecutor, updateOnConflict bool, conflictColumns []string, updateColumns, insertColumns boil.Columns, opts ...UpsertOptionFunc) error {
	if o == nil {
		return errors.New("models: no moderated_channels provided for upsert")
	}
	if !boil.TimestampsAreSkipped(ctx) {
		currTime := time.Now().In(boil.GetLocation())

		if o.CreatedAt.IsZero() {
			o.CreatedAt = currTime
		}
		o.UpdatedAt = currTime
	}

	nzDefaults := queries.NonZeroDefaultSet(moderatedChannelColumnsWithDefault, o)

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

	moderatedChannelUpsertCacheMut.RLock()
	cache, cached := moderatedChannelUpsertCache[key]
	moderatedChannelUpsertCacheMut.RUnlock()

	var err error

	if !cached {
		insert, _ := insertColumns.InsertColumnSet(
			moderatedChannelAllColumns,
			moderatedChannelColumnsWithDefault,
			moderatedChannelColumnsWithoutDefault,
			nzDefaults,
		)

		update := updateColumns.UpdateColumnSet(
			moderatedChannelAllColumns,
			moderatedChannelPrimaryKeyColumns,
		)

		if updateOnConflict && len(update) == 0 {
			return errors.New("models: unable to upsert moderated_channels, could not build update column list")
		}

		ret := strmangle.SetComplement(moderatedChannelAllColumns, strmangle.SetIntersect(insert, update))

		conflict := conflictColumns
		if len(conflict) == 0 && updateOnConflict && len(update) != 0 {
			if len(moderatedChannelPrimaryKeyColumns) == 0 {
				return errors.New("models: unable to upsert moderated_channels, could not build conflict column list")
			}

			conflict = make([]string, len(moderatedChannelPrimaryKeyColumns))
			copy(conflict, moderatedChannelPrimaryKeyColumns)
		}
		cache.query = buildUpsertQueryPostgres(dialect, "\"moderated_channels\"", updateOnConflict, ret, update, conflict, insert, opts...)

		cache.valueMapping, err = queries.BindMapping(moderatedChannelType, moderatedChannelMapping, insert)
		if err != nil {
			return err
		}
		if len(ret) != 0 {
			cache.retMapping, err = queries.BindMapping(moderatedChannelType, moderatedChannelMapping, ret)
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

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, cache.query)
		fmt.Fprintln(writer, vals)
	}
	if len(cache.retMapping) != 0 {
		err = exec.QueryRowContext(ctx, cache.query, vals...).Scan(returns...)
		if errors.Is(err, sql.ErrNoRows) {
			err = nil // Postgres doesn't return anything when there's no update
		}
	} else {
		_, err = exec.ExecContext(ctx, cache.query, vals...)
	}
	if err != nil {
		return errors.Wrap(err, "models: unable to upsert moderated_channels")
	}

	if !cached {
		moderatedChannelUpsertCacheMut.Lock()
		moderatedChannelUpsertCache[key] = cache
		moderatedChannelUpsertCacheMut.Unlock()
	}

	return nil
}

// Delete deletes a single ModeratedChannel record with an executor.
// Delete will match against the primary key column to find the record to delete.
func (o *ModeratedChannel) Delete(ctx context.Context, exec boil.ContextExecutor) error {
	if o == nil {
		return errors.New("models: no ModeratedChannel provided for delete")
	}

	args := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), moderatedChannelPrimaryKeyMapping)
	sql := "DELETE FROM \"moderated_channels\" WHERE \"id\"=$1"

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, args...)
	}
	_, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete from moderated_channels")
	}

	return nil
}

// DeleteAll deletes all matching rows.
func (q moderatedChannelQuery) DeleteAll(ctx context.Context, exec boil.ContextExecutor) error {
	if q.Query == nil {
		return errors.New("models: no moderatedChannelQuery provided for delete all")
	}

	queries.SetDelete(q.Query)

	_, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete all from moderated_channels")
	}

	return nil
}

// DeleteAll deletes all rows in the slice, using an executor.
func (o ModeratedChannelSlice) DeleteAll(ctx context.Context, exec boil.ContextExecutor) error {
	if len(o) == 0 {
		return nil
	}

	var args []interface{}
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), moderatedChannelPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "DELETE FROM \"moderated_channels\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, moderatedChannelPrimaryKeyColumns, len(o))

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, args)
	}
	_, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete all from moderatedChannel slice")
	}

	return nil
}

// Reload refetches the object from the database
// using the primary keys with an executor.
func (o *ModeratedChannel) Reload(ctx context.Context, exec boil.ContextExecutor) error {
	ret, err := FindModeratedChannel(ctx, exec, o.ID)
	if err != nil {
		return err
	}

	*o = *ret
	return nil
}

// ReloadAll refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *ModeratedChannelSlice) ReloadAll(ctx context.Context, exec boil.ContextExecutor) error {
	if o == nil || len(*o) == 0 {
		return nil
	}

	slice := ModeratedChannelSlice{}
	var args []interface{}
	for _, obj := range *o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), moderatedChannelPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "SELECT \"moderated_channels\".* FROM \"moderated_channels\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, moderatedChannelPrimaryKeyColumns, len(*o))

	q := queries.Raw(sql, args...)

	err := q.Bind(ctx, exec, &slice)
	if err != nil {
		return errors.Wrap(err, "models: unable to reload all in ModeratedChannelSlice")
	}

	*o = slice

	return nil
}

// ModeratedChannelExists checks if the ModeratedChannel row exists.
func ModeratedChannelExists(ctx context.Context, exec boil.ContextExecutor, iD int64) (bool, error) {
	var exists bool
	sql := "select exists(select 1 from \"moderated_channels\" where \"id\"=$1 limit 1)"

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, iD)
	}
	row := exec.QueryRowContext(ctx, sql, iD)

	err := row.Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "models: unable to check if moderated_channels exists")
	}

	return exists, nil
}

// Exists checks if the ModeratedChannel row exists.
func (o *ModeratedChannel) Exists(ctx context.Context, exec boil.ContextExecutor) (bool, error) {
	return ModeratedChannelExists(ctx, exec, o.ID)
}
