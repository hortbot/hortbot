// Code generated by SQLBoiler v4.14.2 (https://github.com/volatiletech/sqlboiler). DO NOT EDIT.
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
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"github.com/volatiletech/sqlboiler/v4/queries/qmhelper"
	"github.com/volatiletech/strmangle"
)

// TwitchToken is an object representing the database table.
type TwitchToken struct {
	ID           int64       `boil:"id" json:"id" toml:"id" yaml:"id"`
	CreatedAt    time.Time   `boil:"created_at" json:"created_at" toml:"created_at" yaml:"created_at"`
	UpdatedAt    time.Time   `boil:"updated_at" json:"updated_at" toml:"updated_at" yaml:"updated_at"`
	TwitchID     int64       `boil:"twitch_id" json:"twitch_id" toml:"twitch_id" yaml:"twitch_id"`
	BotName      null.String `boil:"bot_name" json:"bot_name,omitempty" toml:"bot_name" yaml:"bot_name,omitempty"`
	AccessToken  string      `boil:"access_token" json:"access_token" toml:"access_token" yaml:"access_token"`
	TokenType    string      `boil:"token_type" json:"token_type" toml:"token_type" yaml:"token_type"`
	RefreshToken string      `boil:"refresh_token" json:"refresh_token" toml:"refresh_token" yaml:"refresh_token"`
	Expiry       time.Time   `boil:"expiry" json:"expiry" toml:"expiry" yaml:"expiry"`

	R *twitchTokenR `boil:"-" json:"-" toml:"-" yaml:"-"`
	L twitchTokenL  `boil:"-" json:"-" toml:"-" yaml:"-"`
}

var TwitchTokenColumns = struct {
	ID           string
	CreatedAt    string
	UpdatedAt    string
	TwitchID     string
	BotName      string
	AccessToken  string
	TokenType    string
	RefreshToken string
	Expiry       string
}{
	ID:           "id",
	CreatedAt:    "created_at",
	UpdatedAt:    "updated_at",
	TwitchID:     "twitch_id",
	BotName:      "bot_name",
	AccessToken:  "access_token",
	TokenType:    "token_type",
	RefreshToken: "refresh_token",
	Expiry:       "expiry",
}

var TwitchTokenTableColumns = struct {
	ID           string
	CreatedAt    string
	UpdatedAt    string
	TwitchID     string
	BotName      string
	AccessToken  string
	TokenType    string
	RefreshToken string
	Expiry       string
}{
	ID:           "twitch_tokens.id",
	CreatedAt:    "twitch_tokens.created_at",
	UpdatedAt:    "twitch_tokens.updated_at",
	TwitchID:     "twitch_tokens.twitch_id",
	BotName:      "twitch_tokens.bot_name",
	AccessToken:  "twitch_tokens.access_token",
	TokenType:    "twitch_tokens.token_type",
	RefreshToken: "twitch_tokens.refresh_token",
	Expiry:       "twitch_tokens.expiry",
}

// Generated where

var TwitchTokenWhere = struct {
	ID           whereHelperint64
	CreatedAt    whereHelpertime_Time
	UpdatedAt    whereHelpertime_Time
	TwitchID     whereHelperint64
	BotName      whereHelpernull_String
	AccessToken  whereHelperstring
	TokenType    whereHelperstring
	RefreshToken whereHelperstring
	Expiry       whereHelpertime_Time
}{
	ID:           whereHelperint64{field: "\"twitch_tokens\".\"id\""},
	CreatedAt:    whereHelpertime_Time{field: "\"twitch_tokens\".\"created_at\""},
	UpdatedAt:    whereHelpertime_Time{field: "\"twitch_tokens\".\"updated_at\""},
	TwitchID:     whereHelperint64{field: "\"twitch_tokens\".\"twitch_id\""},
	BotName:      whereHelpernull_String{field: "\"twitch_tokens\".\"bot_name\""},
	AccessToken:  whereHelperstring{field: "\"twitch_tokens\".\"access_token\""},
	TokenType:    whereHelperstring{field: "\"twitch_tokens\".\"token_type\""},
	RefreshToken: whereHelperstring{field: "\"twitch_tokens\".\"refresh_token\""},
	Expiry:       whereHelpertime_Time{field: "\"twitch_tokens\".\"expiry\""},
}

// TwitchTokenRels is where relationship names are stored.
var TwitchTokenRels = struct {
}{}

// twitchTokenR is where relationships are stored.
type twitchTokenR struct {
}

// NewStruct creates a new relationship struct
func (*twitchTokenR) NewStruct() *twitchTokenR {
	return &twitchTokenR{}
}

// twitchTokenL is where Load methods for each relationship are stored.
type twitchTokenL struct{}

var (
	twitchTokenAllColumns            = []string{"id", "created_at", "updated_at", "twitch_id", "bot_name", "access_token", "token_type", "refresh_token", "expiry"}
	twitchTokenColumnsWithoutDefault = []string{"twitch_id", "access_token", "token_type", "refresh_token", "expiry"}
	twitchTokenColumnsWithDefault    = []string{"id", "created_at", "updated_at", "bot_name"}
	twitchTokenPrimaryKeyColumns     = []string{"id"}
	twitchTokenGeneratedColumns      = []string{}
)

type (
	// TwitchTokenSlice is an alias for a slice of pointers to TwitchToken.
	// This should almost always be used instead of []TwitchToken.
	TwitchTokenSlice []*TwitchToken

	twitchTokenQuery struct {
		*queries.Query
	}
)

// Cache for insert, update and upsert
var (
	twitchTokenType                 = reflect.TypeOf(&TwitchToken{})
	twitchTokenMapping              = queries.MakeStructMapping(twitchTokenType)
	twitchTokenPrimaryKeyMapping, _ = queries.BindMapping(twitchTokenType, twitchTokenMapping, twitchTokenPrimaryKeyColumns)
	twitchTokenInsertCacheMut       sync.RWMutex
	twitchTokenInsertCache          = make(map[string]insertCache)
	twitchTokenUpdateCacheMut       sync.RWMutex
	twitchTokenUpdateCache          = make(map[string]updateCache)
	twitchTokenUpsertCacheMut       sync.RWMutex
	twitchTokenUpsertCache          = make(map[string]insertCache)
)

var (
	// Force time package dependency for automated UpdatedAt/CreatedAt.
	_ = time.Second
	// Force qmhelper dependency for where clause generation (which doesn't
	// always happen)
	_ = qmhelper.Where
)

// One returns a single twitchToken record from the query.
func (q twitchTokenQuery) One(ctx context.Context, exec boil.ContextExecutor) (*TwitchToken, error) {
	o := &TwitchToken{}

	queries.SetLimit(q.Query, 1)

	err := q.Bind(ctx, exec, o)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: failed to execute a one query for twitch_tokens")
	}

	return o, nil
}

// All returns all TwitchToken records from the query.
func (q twitchTokenQuery) All(ctx context.Context, exec boil.ContextExecutor) (TwitchTokenSlice, error) {
	var o []*TwitchToken

	err := q.Bind(ctx, exec, &o)
	if err != nil {
		return nil, errors.Wrap(err, "models: failed to assign all query results to TwitchToken slice")
	}

	return o, nil
}

// Count returns the count of all TwitchToken records in the query.
func (q twitchTokenQuery) Count(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to count twitch_tokens rows")
	}

	return count, nil
}

// Exists checks if the row exists in the table.
func (q twitchTokenQuery) Exists(ctx context.Context, exec boil.ContextExecutor) (bool, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)
	queries.SetLimit(q.Query, 1)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "models: failed to check if twitch_tokens exists")
	}

	return count > 0, nil
}

// TwitchTokens retrieves all the records using an executor.
func TwitchTokens(mods ...qm.QueryMod) twitchTokenQuery {
	mods = append(mods, qm.From("\"twitch_tokens\""))
	q := NewQuery(mods...)
	if len(queries.GetSelect(q)) == 0 {
		queries.SetSelect(q, []string{"\"twitch_tokens\".*"})
	}

	return twitchTokenQuery{q}
}

// FindTwitchToken retrieves a single record by ID with an executor.
// If selectCols is empty Find will return all columns.
func FindTwitchToken(ctx context.Context, exec boil.ContextExecutor, iD int64, selectCols ...string) (*TwitchToken, error) {
	twitchTokenObj := &TwitchToken{}

	sel := "*"
	if len(selectCols) > 0 {
		sel = strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, selectCols), ",")
	}
	query := fmt.Sprintf(
		"select %s from \"twitch_tokens\" where \"id\"=$1", sel,
	)

	q := queries.Raw(query, iD)

	err := q.Bind(ctx, exec, twitchTokenObj)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: unable to select from twitch_tokens")
	}

	return twitchTokenObj, nil
}

// Insert a single record using an executor.
// See boil.Columns.InsertColumnSet documentation to understand column list inference for inserts.
func (o *TwitchToken) Insert(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) error {
	if o == nil {
		return errors.New("models: no twitch_tokens provided for insertion")
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

	nzDefaults := queries.NonZeroDefaultSet(twitchTokenColumnsWithDefault, o)

	key := makeCacheKey(columns, nzDefaults)
	twitchTokenInsertCacheMut.RLock()
	cache, cached := twitchTokenInsertCache[key]
	twitchTokenInsertCacheMut.RUnlock()

	if !cached {
		wl, returnColumns := columns.InsertColumnSet(
			twitchTokenAllColumns,
			twitchTokenColumnsWithDefault,
			twitchTokenColumnsWithoutDefault,
			nzDefaults,
		)

		cache.valueMapping, err = queries.BindMapping(twitchTokenType, twitchTokenMapping, wl)
		if err != nil {
			return err
		}
		cache.retMapping, err = queries.BindMapping(twitchTokenType, twitchTokenMapping, returnColumns)
		if err != nil {
			return err
		}
		if len(wl) != 0 {
			cache.query = fmt.Sprintf("INSERT INTO \"twitch_tokens\" (\"%s\") %%sVALUES (%s)%%s", strings.Join(wl, "\",\""), strmangle.Placeholders(dialect.UseIndexPlaceholders, len(wl), 1, 1))
		} else {
			cache.query = "INSERT INTO \"twitch_tokens\" %sDEFAULT VALUES%s"
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
		return errors.Wrap(err, "models: unable to insert into twitch_tokens")
	}

	if !cached {
		twitchTokenInsertCacheMut.Lock()
		twitchTokenInsertCache[key] = cache
		twitchTokenInsertCacheMut.Unlock()
	}

	return nil
}

// Update uses an executor to update the TwitchToken.
// See boil.Columns.UpdateColumnSet documentation to understand column list inference for updates.
// Update does not automatically update the record in case of default values. Use .Reload() to refresh the records.
func (o *TwitchToken) Update(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) error {
	if !boil.TimestampsAreSkipped(ctx) {
		currTime := time.Now().In(boil.GetLocation())

		o.UpdatedAt = currTime
	}

	var err error
	key := makeCacheKey(columns, nil)
	twitchTokenUpdateCacheMut.RLock()
	cache, cached := twitchTokenUpdateCache[key]
	twitchTokenUpdateCacheMut.RUnlock()

	if !cached {
		wl := columns.UpdateColumnSet(
			twitchTokenAllColumns,
			twitchTokenPrimaryKeyColumns,
		)

		if !columns.IsWhitelist() {
			wl = strmangle.SetComplement(wl, []string{"created_at"})
		}
		if len(wl) == 0 {
			return errors.New("models: unable to update twitch_tokens, could not build whitelist")
		}

		cache.query = fmt.Sprintf("UPDATE \"twitch_tokens\" SET %s WHERE %s",
			strmangle.SetParamNames("\"", "\"", 1, wl),
			strmangle.WhereClause("\"", "\"", len(wl)+1, twitchTokenPrimaryKeyColumns),
		)
		cache.valueMapping, err = queries.BindMapping(twitchTokenType, twitchTokenMapping, append(wl, twitchTokenPrimaryKeyColumns...))
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
		return errors.Wrap(err, "models: unable to update twitch_tokens row")
	}

	if !cached {
		twitchTokenUpdateCacheMut.Lock()
		twitchTokenUpdateCache[key] = cache
		twitchTokenUpdateCacheMut.Unlock()
	}

	return nil
}

// UpdateAll updates all rows with the specified column values.
func (q twitchTokenQuery) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) error {
	queries.SetUpdate(q.Query, cols)

	_, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return errors.Wrap(err, "models: unable to update all for twitch_tokens")
	}

	return nil
}

// UpdateAll updates all rows with the specified column values, using an executor.
func (o TwitchTokenSlice) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) error {
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
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), twitchTokenPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf("UPDATE \"twitch_tokens\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, colNames),
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), len(colNames)+1, twitchTokenPrimaryKeyColumns, len(o)))

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, args...)
	}
	_, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to update all in twitchToken slice")
	}

	return nil
}

// Upsert attempts an insert using an executor, and does an update or ignore on conflict.
// See boil.Columns documentation for how to properly use updateColumns and insertColumns.
func (o *TwitchToken) Upsert(ctx context.Context, exec boil.ContextExecutor, updateOnConflict bool, conflictColumns []string, updateColumns, insertColumns boil.Columns) error {
	if o == nil {
		return errors.New("models: no twitch_tokens provided for upsert")
	}
	if !boil.TimestampsAreSkipped(ctx) {
		currTime := time.Now().In(boil.GetLocation())

		if o.CreatedAt.IsZero() {
			o.CreatedAt = currTime
		}
		o.UpdatedAt = currTime
	}

	nzDefaults := queries.NonZeroDefaultSet(twitchTokenColumnsWithDefault, o)

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

	twitchTokenUpsertCacheMut.RLock()
	cache, cached := twitchTokenUpsertCache[key]
	twitchTokenUpsertCacheMut.RUnlock()

	var err error

	if !cached {
		insert, ret := insertColumns.InsertColumnSet(
			twitchTokenAllColumns,
			twitchTokenColumnsWithDefault,
			twitchTokenColumnsWithoutDefault,
			nzDefaults,
		)

		update := updateColumns.UpdateColumnSet(
			twitchTokenAllColumns,
			twitchTokenPrimaryKeyColumns,
		)

		if updateOnConflict && len(update) == 0 {
			return errors.New("models: unable to upsert twitch_tokens, could not build update column list")
		}

		conflict := conflictColumns
		if len(conflict) == 0 {
			conflict = make([]string, len(twitchTokenPrimaryKeyColumns))
			copy(conflict, twitchTokenPrimaryKeyColumns)
		}
		cache.query = buildUpsertQueryPostgres(dialect, "\"twitch_tokens\"", updateOnConflict, ret, update, conflict, insert)

		cache.valueMapping, err = queries.BindMapping(twitchTokenType, twitchTokenMapping, insert)
		if err != nil {
			return err
		}
		if len(ret) != 0 {
			cache.retMapping, err = queries.BindMapping(twitchTokenType, twitchTokenMapping, ret)
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
		return errors.Wrap(err, "models: unable to upsert twitch_tokens")
	}

	if !cached {
		twitchTokenUpsertCacheMut.Lock()
		twitchTokenUpsertCache[key] = cache
		twitchTokenUpsertCacheMut.Unlock()
	}

	return nil
}

// Delete deletes a single TwitchToken record with an executor.
// Delete will match against the primary key column to find the record to delete.
func (o *TwitchToken) Delete(ctx context.Context, exec boil.ContextExecutor) error {
	if o == nil {
		return errors.New("models: no TwitchToken provided for delete")
	}

	args := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), twitchTokenPrimaryKeyMapping)
	sql := "DELETE FROM \"twitch_tokens\" WHERE \"id\"=$1"

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, args...)
	}
	_, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete from twitch_tokens")
	}

	return nil
}

// DeleteAll deletes all matching rows.
func (q twitchTokenQuery) DeleteAll(ctx context.Context, exec boil.ContextExecutor) error {
	if q.Query == nil {
		return errors.New("models: no twitchTokenQuery provided for delete all")
	}

	queries.SetDelete(q.Query)

	_, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete all from twitch_tokens")
	}

	return nil
}

// DeleteAll deletes all rows in the slice, using an executor.
func (o TwitchTokenSlice) DeleteAll(ctx context.Context, exec boil.ContextExecutor) error {
	if len(o) == 0 {
		return nil
	}

	var args []interface{}
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), twitchTokenPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "DELETE FROM \"twitch_tokens\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, twitchTokenPrimaryKeyColumns, len(o))

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, args)
	}
	_, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "models: unable to delete all from twitchToken slice")
	}

	return nil
}

// Reload refetches the object from the database
// using the primary keys with an executor.
func (o *TwitchToken) Reload(ctx context.Context, exec boil.ContextExecutor) error {
	ret, err := FindTwitchToken(ctx, exec, o.ID)
	if err != nil {
		return err
	}

	*o = *ret
	return nil
}

// ReloadAll refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *TwitchTokenSlice) ReloadAll(ctx context.Context, exec boil.ContextExecutor) error {
	if o == nil || len(*o) == 0 {
		return nil
	}

	slice := TwitchTokenSlice{}
	var args []interface{}
	for _, obj := range *o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), twitchTokenPrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "SELECT \"twitch_tokens\".* FROM \"twitch_tokens\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, twitchTokenPrimaryKeyColumns, len(*o))

	q := queries.Raw(sql, args...)

	err := q.Bind(ctx, exec, &slice)
	if err != nil {
		return errors.Wrap(err, "models: unable to reload all in TwitchTokenSlice")
	}

	*o = slice

	return nil
}

// TwitchTokenExists checks if the TwitchToken row exists.
func TwitchTokenExists(ctx context.Context, exec boil.ContextExecutor, iD int64) (bool, error) {
	var exists bool
	sql := "select exists(select 1 from \"twitch_tokens\" where \"id\"=$1 limit 1)"

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, iD)
	}
	row := exec.QueryRowContext(ctx, sql, iD)

	err := row.Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "models: unable to check if twitch_tokens exists")
	}

	return exists, nil
}

// Exists checks if the TwitchToken row exists.
func (o *TwitchToken) Exists(ctx context.Context, exec boil.ContextExecutor) (bool, error) {
	return TwitchTokenExists(ctx, exec, o.ID)
}
