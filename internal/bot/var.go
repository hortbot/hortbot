package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/jackc/pgx/v5"
)

func (s *session) VarGet(ctx context.Context, name string) (string, bool, error) {
	v, err := s.Queries.GetVariable(ctx, dbsql.GetVariableParams{
		ChannelID: s.Channel.ID,
		Name:      name,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}

	if err != nil {
		return "", false, fmt.Errorf("getting variable: %w", err)
	}

	return v.Value, true, nil
}

func (s *session) VarGetByChannel(ctx context.Context, ch, name string) (string, bool, error) {
	v, err := s.Queries.GetVariableByChannelName(ctx, dbsql.GetVariableByChannelNameParams{
		ChannelName: strings.ToLower(ch),
		Name:        name,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}

	if err != nil {
		return "", false, fmt.Errorf("getting variable: %w", err)
	}

	return v.Value, true, nil
}

func (s *session) VarSet(ctx context.Context, name, value string) error {
	err := s.Queries.UpsertVariable(ctx, dbsql.UpsertVariableParams{
		ChannelID: s.Channel.ID,
		Name:      name,
		Value:     value,
	})
	if err != nil {
		return fmt.Errorf("upserting variable: %w", err)
	}

	return nil
}

func (s *session) VarDelete(ctx context.Context, name string) error {
	if err := s.Queries.DeleteVariable(ctx, dbsql.DeleteVariableParams{
		ChannelID: s.Channel.ID,
		Name:      name,
	}); err != nil {
		return fmt.Errorf("deleting variable: %w", err)
	}

	return nil
}

func (s *session) VarIncrement(ctx context.Context, name string, inc int64) (n int64, badVar bool, err error) {
	// TODO: Do this in a psql query, not in Go.

	v, err := s.Queries.GetVariable(ctx, dbsql.GetVariableParams{
		ChannelID: s.Channel.ID,
		Name:      name,
	})

	if errors.Is(err, pgx.ErrNoRows) {
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

	if err := s.Queries.UpdateVariableValue(ctx, dbsql.UpdateVariableValueParams{
		Value: v.Value,
		ID:    v.ID,
	}); err != nil {
		return 0, false, fmt.Errorf("updating variable: %w", err)
	}

	return vInt, false, nil
}
