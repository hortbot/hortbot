package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries"
	"go.opencensus.io/trace"
)

func (s *session) VarGet(ctx context.Context, name string) (string, bool, error) {
	ctx, span := trace.StartSpan(ctx, "VarGet")
	defer span.End()

	v, err := s.Channel.Variables(models.VariableWhere.Name.EQ(name)).One(ctx, s.Tx)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}

	if err != nil {
		return "", false, err
	}

	return v.Value, true, nil
}

func (s *session) VarGetByChannel(ctx context.Context, ch, name string) (string, bool, error) {
	ctx, span := trace.StartSpan(ctx, "VarGetByChannel")
	defer span.End()

	var v models.Variable

	err := queries.Raw(`
		SELECT variables.*
		FROM variables
		JOIN channels ON channels.id = variables.channel_id
		WHERE channels.name = $1 AND variables.name = $2`,
		strings.ToLower(ch), name).Bind(ctx, s.Tx, &v)

	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}

	if err != nil {
		return "", false, fmt.Errorf("getting variable: %w", err)
	}

	return v.Value, true, nil
}

var varConflictCols = []string{
	models.VariableColumns.ChannelID,
	models.VariableColumns.Name,
}

func (s *session) VarSet(ctx context.Context, name, value string) error {
	ctx, span := trace.StartSpan(ctx, "VarSet")
	defer span.End()

	v := &models.Variable{
		ChannelID: s.Channel.ID,
		Name:      name,
		Value:     value,
	}

	return v.Upsert(ctx, s.Tx, true, varConflictCols, boil.Blacklist(models.VariableTableColumns.CreatedAt), boil.Infer())
}

func (s *session) VarDelete(ctx context.Context, name string) error {
	ctx, span := trace.StartSpan(ctx, "VarDelete")
	defer span.End()

	return s.Channel.Variables(models.VariableWhere.Name.EQ(name)).DeleteAll(ctx, s.Tx)
}

func (s *session) VarIncrement(ctx context.Context, name string, inc int64) (n int64, badVar bool, err error) {
	ctx, span := trace.StartSpan(ctx, "VarIncrement")
	defer span.End()

	// TODO: Do this in a psql query, not in Go.

	v, err := s.Channel.Variables(models.VariableWhere.Name.EQ(name)).One(ctx, s.Tx)

	if errors.Is(err, sql.ErrNoRows) {
		return inc, false, s.VarSet(ctx, name, strconv.FormatInt(inc, 10))
	}

	if err != nil {
		return 0, false, err
	}

	vInt, err := strconv.ParseInt(v.Value, 10, 64)
	if err != nil {
		return 0, true, nil
	}

	vInt += inc

	v.Value = strconv.FormatInt(vInt, 10)

	if err := v.Update(ctx, s.Tx, boil.Whitelist(models.VariableColumns.UpdatedAt, models.VariableColumns.Value)); err != nil {
		return 0, false, err
	}

	return vInt, false, nil
}
