package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries"
	"github.com/hortbot/hortbot/internal/db/models"
)

func (s *session) VarGet(ctx context.Context, name string) (string, bool, error) {
	v, err := s.Channel.Variables(models.VariableWhere.Name.EQ(name)).One(ctx, s.Tx)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}

	if err != nil {
		return "", false, fmt.Errorf("getting variable: %w", err)
	}

	return v.Value, true, nil
}

func (s *session) VarGetByChannel(ctx context.Context, ch, name string) (string, bool, error) {
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
	v := &models.Variable{
		ChannelID: s.Channel.ID,
		Name:      name,
		Value:     value,
	}

	if err := v.Upsert(ctx, s.Tx, true, varConflictCols, boil.Blacklist(models.VariableTableColumns.CreatedAt), boil.Infer()); err != nil {
		return fmt.Errorf("upserting variable: %w", err)
	}

	return nil
}

func (s *session) VarDelete(ctx context.Context, name string) error {
	if err := s.Channel.Variables(models.VariableWhere.Name.EQ(name)).DeleteAll(ctx, s.Tx); err != nil {
		return fmt.Errorf("deleting variable: %w", err)
	}

	return nil
}

func (s *session) VarIncrement(ctx context.Context, name string, inc int64) (n int64, badVar bool, err error) {
	// TODO: Do this in a psql query, not in Go.

	v, err := s.Channel.Variables(models.VariableWhere.Name.EQ(name)).One(ctx, s.Tx)

	if errors.Is(err, sql.ErrNoRows) {
		return inc, false, s.VarSet(ctx, name, strconv.FormatInt(inc, 10))
	}

	if err != nil {
		return 0, false, fmt.Errorf("getting variable: %w", err)
	}

	vInt, err := strconv.ParseInt(v.Value, 10, 64)
	if err != nil {
		return 0, true, nil //nolint:nilerr
	}

	vInt += inc

	v.Value = strconv.FormatInt(vInt, 10)

	if err := v.Update(ctx, s.Tx, boil.Whitelist(models.VariableColumns.UpdatedAt, models.VariableColumns.Value)); err != nil {
		return 0, false, fmt.Errorf("updating variable: %w", err)
	}

	return vInt, false, nil
}
