package bot

import (
	"context"
	"database/sql"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/boil"
)

func cmdSimpleCommand(ctx context.Context, s *Session, args string) error {
	args = strings.TrimSpace(args)

	usage := func() error {
		return s.ReplyUsage("command add|delete|restrict ...")
	}

	subcommand, args := splitSpace(args)

	switch subcommand {
	case "add":
		usage := func() error {
			return s.ReplyUsage("command add <name> <text>")
		}

		name, text := splitSpace(args)

		if name == "" || text == "" {
			return usage()
		}

		name = strings.ToLower(name)

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
			command.Editor = s.User
			if err := command.Update(ctx, s.Tx, boil.Whitelist(models.SimpleCommandColumns.Message, models.SimpleCommandColumns.Editor)); err != nil {
				return err
			}

			return s.Replyf("command %s updated", name)
		}

		command = &models.SimpleCommand{
			Name:        name,
			ChannelID:   s.Channel.ID,
			Message:     text,
			AccessLevel: models.AccessLevelSubscriber,
			Editor:      s.User,
		}

		if err := command.Insert(ctx, s.Tx, boil.Infer()); err != nil {
			return err
		}

		return s.Replyf("command %s added, restricted to %s and above", name, flect.Pluralize(command.AccessLevel))

	case "delete":
		usage := func() error {
			return s.ReplyUsage("command delete <name>")
		}

		if args == "" {
			return usage()
		}

		args = strings.ToLower(args)

		command, err := models.SimpleCommands(
			models.SimpleCommandWhere.ChannelID.EQ(s.Channel.ID),
			models.SimpleCommandWhere.Name.EQ(args),
		).One(ctx, s.Tx)

		if err == sql.ErrNoRows {
			return s.Replyf("command %s not found", args)
		}

		if err != nil {
			return err
		}

		err = command.Delete(ctx, s.Tx)
		if err != nil {
			return err
		}

		return s.Replyf("command %s deleted", args)

	case "restrict":
		usage := func() error {
			return s.ReplyUsage("command restrict <name> everyone|regulars|subs|mods|broadcaster|admin")
		}

		name, level := splitSpace(args)

		if name == "" {
			return usage()
		}

		command, err := models.SimpleCommands(
			models.SimpleCommandWhere.ChannelID.EQ(s.Channel.ID),
			models.SimpleCommandWhere.Name.EQ(name),
		).One(ctx, s.Tx)

		if err == sql.ErrNoRows {
			return s.Replyf("command %s not found", name)
		}

		if err != nil {
			return err
		}

		if level == "" {
			return s.Replyf("command %s is restricted to %s and above", name, flect.Pluralize(command.AccessLevel))
		}

		level = strings.ToLower(level)

		var newLevel string
		switch level {
		case "everyone", "all", "everybody", "normal":
			newLevel = models.AccessLevelEveryone
		case "default", "sub", "subs", "subscriber", "subscrbers", "regular", "regulars", "reg", "regs":
			newLevel = models.AccessLevelSubscriber
		case "mod", "mods", "moderator", "moderators":
			newLevel = models.AccessLevelModerator
		case "broadcaster", "broadcasters", "owner", "owners", "streamer", "streamers":
			newLevel = models.AccessLevelBroadcaster
		case "admin", "admins":
			newLevel = models.AccessLevelAdmin
		default:
			return usage()
		}

		command.AccessLevel = newLevel
		command.Editor = s.User

		if err := command.Update(ctx, s.Tx, boil.Whitelist(models.SimpleCommandColumns.AccessLevel, models.SimpleCommandColumns.Editor)); err != nil {
			return err
		}

		return s.Replyf("command %s restricted to %s and above", name, flect.Pluralize(command.AccessLevel))

	case "editor":
		return errNotImplemented

	default:
		return usage()
	}
}

func init() {
	flect.AddPlural("everyone", "everyone")
}
