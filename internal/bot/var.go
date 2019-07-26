package bot

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/boil"
)

func (s *session) VarGet(ctx context.Context, name string) (string, bool, error) {
	v, err := s.Channel.Variables(models.VariableWhere.Name.EQ(name)).One(ctx, s.Tx)
	if err == sql.ErrNoRows {
		return "", false, nil
	}

	if err != nil {
		return "", false, err
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

	return v.Upsert(ctx, s.Tx, true, varConflictCols, boil.Infer(), boil.Infer())
}

func (s *session) VarDelete(ctx context.Context, name string) error {
	return s.Channel.Variables(models.VariableWhere.Name.EQ(name)).DeleteAll(ctx, s.Tx)
}

func (s *session) VarIncrement(ctx context.Context, name string, inc int64) (n int64, badVar bool, err error) {
	v, err := s.Channel.Variables(models.VariableWhere.Name.EQ(name)).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
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
