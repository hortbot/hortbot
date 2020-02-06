// Package sitedb reads out channel and variable information from CoeBot site
// DB dumps.
package sitedb

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
)

type Channel struct {
	DisplayName              string `json:"display_name"`
	BotName                  string `json:"bot_name"`
	YouTube                  string `json:"youtube"`
	Twitter                  string `json:"twitter"`
	Active                   bool   `json:"active"`
	ShouldShowOffensiveWords bool   `json:"should_show_offensive_words"`
	ShouldShowBoir           bool   `json:"should_show_boir"`
}

func Channels(ctx context.Context, db *sqlx.DB) (map[string]*Channel, error) {
	rows, err := db.Unsafe().QueryxContext(ctx, "SELECT * FROM site_channels")
	if err != nil {
		return nil, err
	}

	channels := make(map[string]*Channel)

	for rows.Next() {
		var row struct {
			Channel                  string        `db:"channel"`
			DisplayName              string        `db:"displayName"`
			BotName                  string        `db:"botChannel"`
			YouTube                  string        `db:"youtube"`
			Twitter                  string        `db:"twitter"`
			Active                   types.BitBool `db:"isActive"`
			ShouldShowOffensiveWords types.BitBool `db:"shouldShowOffensiveWords"`
			ShouldShowBoir           types.BitBool `db:"shouldShowBoir"`
		}

		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}

		channels[row.Channel] = &Channel{
			DisplayName:              row.DisplayName,
			BotName:                  row.BotName,
			YouTube:                  row.YouTube,
			Twitter:                  row.Twitter,
			Active:                   bool(row.Active),
			ShouldShowOffensiveWords: bool(row.ShouldShowOffensiveWords),
			ShouldShowBoir:           bool(row.ShouldShowBoir),
		}
	}

	return channels, nil
}

type Var struct {
	Name         string    `json:"name"`
	Value        string    `json:"value"`
	Description  string    `json:"description"`
	LastModified time.Time `json:"last_modified"`
}

func Vars(ctx context.Context, db *sqlx.DB) (map[string][]*Var, error) {
	rows, err := db.Unsafe().QueryxContext(ctx, "SELECT * FROM site_vars")
	if err != nil {
		return nil, err
	}

	vars := make(map[string][]*Var)

	for rows.Next() {
		var row struct {
			Channel      string `db:"channel"`
			Name         string `db:"var"`
			Value        string `db:"value"`
			Description  string `db:"description"`
			LastModified int64  `db:"lastModified"`
		}

		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}

		v := &Var{
			Name:         row.Name,
			Value:        row.Value,
			Description:  row.Description,
			LastModified: time.Unix(row.LastModified, 0),
		}

		vars[row.Channel] = append(vars[row.Channel], v)
	}

	return vars, nil
}
