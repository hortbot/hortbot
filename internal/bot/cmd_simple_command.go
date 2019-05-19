package bot

import (
	"context"
	"database/sql"
	"strings"

	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/boil"
)

func cmdSimpleCommand(ctx context.Context, s *Session, args string) error {
	args = strings.TrimSpace(args)

	usage := func() error {
		return s.Replyf("usage: %scommand add|delete|restrict ...", s.Channel.Prefix)
	}

	subcommand, args := splitSpace(args)

	switch subcommand {
	case "":
		return usage()

	case "add":
		usage := func() error {
			return s.Replyf("usage: %scommand add <name> <text>", s.Channel.Prefix)
		}

		name, text := splitSpace(args)

		if name == "" || text == "" {
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
			if err := command.Update(ctx, s.Tx, boil.Whitelist(models.SimpleCommandColumns.Message)); err != nil {
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
