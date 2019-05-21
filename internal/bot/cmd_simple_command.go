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

func cmdSimpleCommand(ctx context.Context, s *Session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("command add|delete|restrict ...")
	}

	args = strings.TrimSpace(args)
	subcommand, args := splitSpace(args)

	switch subcommand {
	case "add": // TODO: variants of add (adda, adds, addm, addb) which preset the access level.
		return cmdSimpleCommandAdd(ctx, s, args, models.AccessLevelSubscriber)

	case "delete":
		return cmdSimpleCommandDelete(ctx, s, args)

	case "restrict":
		return cmdSimpleCommandRestrict(ctx, s, args)

	case "editor":
		return errNotImplemented

	default:
		return usage()
	}
}

func cmdSimpleCommandAdd(ctx context.Context, s *Session, args string, level string) error {
	usage := func() error {
		return s.ReplyUsage("command add <name> <text>")
	}

	name, text := splitSpace(args)

	if name == "" || text == "" {
		return usage()
	}

	name = strings.ToLower(name)

	if reservedCommandNames[name] {
		return s.Replyf("command name %s is reserved", name)
	}

	var warning string
	if _, ok := builtinCommands[name]; ok {
		warning = "; warning: " + name + " is a builtin command and will now only be accessible via " + s.Channel.Prefix + "builtin " + name
	}

	_, err := cbp.Parse(text)
	if err != nil {
		return s.Replyf("error parsing command%s", warning)
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

		return s.Replyf("command %s updated%s", name, warning)
	}

	command = &models.SimpleCommand{
		Name:        name,
		ChannelID:   s.Channel.ID,
		Message:     text,
		AccessLevel: level,
		Editor:      s.User,
	}

	if err := command.Insert(ctx, s.Tx, boil.Infer()); err != nil {
		return err
	}

	return s.Replyf("command %s added, restricted to %s and above%s", name, flect.Pluralize(command.AccessLevel), warning)
}

func cmdSimpleCommandDelete(ctx context.Context, s *Session, args string) error {
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
}

func cmdSimpleCommandRestrict(ctx context.Context, s *Session, args string) error {
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
}

func init() {
	flect.AddPlural("everyone", "everyone")
}
