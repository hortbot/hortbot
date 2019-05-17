package bot

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/boil"
)

var errNotImplemented = errors.New("not implemented")

var builtins = map[string]func(ctx context.Context, s *Session, args string) error{
	"command": simpleCommand,
}

func simpleCommand(ctx context.Context, s *Session, args string) error {
	usage := func() error {
		return s.Replyf("usage: %scommand add|delete|restrict ...", s.Channel.Prefix)
	}

	params := strings.SplitN(args, " ", 2)

	if len(params) == 0 {
		return usage()
	}

	subcommand := params[0]

	if len(params) == 2 {
		args = params[1]
	} else {
		args = ""
	}

	switch subcommand {
	case "add":
		usage := func() error {
			return s.Replyf("usage: %scommand add <name> <text>", s.Channel.Prefix)
		}

		params := strings.SplitN(args, " ", 2)
		if len(params) != 2 {
			return usage()
		}

		name := params[0]
		text := strings.TrimSpace(params[1])

		if text == "" {
			return usage()
		}

		_, err := cbp.Parse(text)
		if err != nil {
			return s.Replyf("error parsing command")
		}

		command, err := models.SimpleCommands(
			models.SimpleCommandWhere.ChannelID.EQ(s.Channel.ID),
			models.SimpleCommandWhere.Name.EQ(name),
		).One(ctx, s.Tx)

		update := err != sql.ErrNoRows

		if err != nil && err != sql.ErrNoRows {
			return err
		}

		if update {
			command.Message = text
			if err := command.Update(ctx, s.Tx, boil.Infer()); err != nil {
				return err
			}

			return s.Replyf("command %s updated", name)
		}

		command = &models.SimpleCommand{
			Name:      name,
			ChannelID: s.Channel.ID,
			Message:   text,
		}

		if err := command.Insert(ctx, s.Tx, boil.Infer()); err != nil {
			return err
		}

		return s.Replyf("command %s added", name)

	case "delete":
		return errNotImplemented

	case "restrict":
		return errNotImplemented

	default:
		return usage()
	}
}
