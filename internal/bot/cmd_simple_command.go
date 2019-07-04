package bot

import (
	"context"
	"database/sql"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

var scCommands handlerMap = map[string]handlerFunc{
	"add":             {fn: cmdSimpleCommandAddNormal, minLevel: levelModerator},
	"addb":            {fn: cmdSimpleCommandAddBroadcaster, minLevel: levelModerator},
	"addbroadcaster":  {fn: cmdSimpleCommandAddBroadcaster, minLevel: levelModerator},
	"addbroadcasters": {fn: cmdSimpleCommandAddBroadcaster, minLevel: levelModerator},
	"addo":            {fn: cmdSimpleCommandAddBroadcaster, minLevel: levelModerator},
	"addowner":        {fn: cmdSimpleCommandAddBroadcaster, minLevel: levelModerator},
	"addowners":       {fn: cmdSimpleCommandAddBroadcaster, minLevel: levelModerator},
	"addstreamer":     {fn: cmdSimpleCommandAddBroadcaster, minLevel: levelModerator},
	"addstreamers":    {fn: cmdSimpleCommandAddBroadcaster, minLevel: levelModerator},
	"addm":            {fn: cmdSimpleCommandAddModerator, minLevel: levelModerator},
	"addmod":          {fn: cmdSimpleCommandAddModerator, minLevel: levelModerator},
	"addmods":         {fn: cmdSimpleCommandAddModerator, minLevel: levelModerator},
	"adds":            {fn: cmdSimpleCommandAddSubscriber, minLevel: levelModerator},
	"addsub":          {fn: cmdSimpleCommandAddSubscriber, minLevel: levelModerator},
	"addsubs":         {fn: cmdSimpleCommandAddSubscriber, minLevel: levelModerator},
	"adde":            {fn: cmdSimpleCommandAddEveryone, minLevel: levelModerator},
	"adda":            {fn: cmdSimpleCommandAddEveryone, minLevel: levelModerator},
	"addeveryone":     {fn: cmdSimpleCommandAddEveryone, minLevel: levelModerator},
	"addall":          {fn: cmdSimpleCommandAddEveryone, minLevel: levelModerator},
	"delete":          {fn: cmdSimpleCommandDelete, minLevel: levelModerator},
	"remove":          {fn: cmdSimpleCommandDelete, minLevel: levelModerator},
	"restrict":        {fn: cmdSimpleCommandRestrict, minLevel: levelModerator},
	"editor":          {fn: cmdSimpleCommandProperty, minLevel: levelModerator},
	"author":          {fn: cmdSimpleCommandProperty, minLevel: levelModerator},
	"count":           {fn: cmdSimpleCommandProperty, minLevel: levelModerator},
	"rename":          {fn: cmdSimpleCommandRename, minLevel: levelModerator},
	"get":             {fn: cmdSimpleCommandGet, minLevel: levelModerator},
	// TODO: clone
}

func cmdSimpleCommand(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	ok, err := scCommands.run(ctx, s, subcommand, args)
	if err != nil {
		return err
	}

	if !ok {
		return s.ReplyUsage("add|delete|restrict|...")
	}

	return nil
}

func cmdSimpleCommandAddNormal(ctx context.Context, s *session, cmd string, args string) error {
	return cmdSimpleCommandAdd(ctx, s, args, levelSubscriber, false)
}

func cmdSimpleCommandAddBroadcaster(ctx context.Context, s *session, cmd string, args string) error {
	return cmdSimpleCommandAdd(ctx, s, args, levelBroadcaster, true)
}

func cmdSimpleCommandAddModerator(ctx context.Context, s *session, cmd string, args string) error {
	return cmdSimpleCommandAdd(ctx, s, args, levelModerator, true)
}

func cmdSimpleCommandAddSubscriber(ctx context.Context, s *session, cmd string, args string) error {
	return cmdSimpleCommandAdd(ctx, s, args, levelSubscriber, true)
}

func cmdSimpleCommandAddEveryone(ctx context.Context, s *session, cmd string, args string) error {
	return cmdSimpleCommandAdd(ctx, s, args, levelEveryone, true)
}

func cmdSimpleCommandAdd(ctx context.Context, s *session, args string, level accessLevel, forceLevel bool) error {
	usage := func() error {
		return s.ReplyUsage("<name> <text>")
	}

	name, text := splitSpace(args)

	if name == "" || text == "" {
		return usage()
	}

	name = strings.ToLower(name)

	if reservedCommandNames[name] {
		return s.Replyf("Command name '%s' is reserved.", name)
	}

	// TODO: remove this warning
	var warning string
	if _, ok := builtinCommands[name]; ok {
		warning = " Warning: '" + name + "' is a builtin command and will now only be accessible via " + s.Channel.Prefix + "builtin " + name
	}

	_, err := cbp.Parse(text)
	if err != nil {
		return s.Replyf("Error parsing command.%s", warning)
	}

	command, err := models.SimpleCommands(
		models.SimpleCommandWhere.ChannelID.EQ(s.Channel.ID),
		models.SimpleCommandWhere.Name.EQ(name),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)

	if err != nil && err != sql.ErrNoRows {
		return err
	}

	update := err != sql.ErrNoRows

	if !s.UserLevel.CanAccess(level) {
		a := "add"
		if update {
			a = "update"
		}

		return s.Replyf("Your level is %s; you cannot %s a command with level %s.", s.UserLevel.PGEnum(), a, level.PGEnum())
	}

	if update {
		if !s.UserLevel.CanAccess(newAccessLevel(command.AccessLevel)) {
			al := flect.Pluralize(command.AccessLevel)
			return s.Replyf("Command '%s' is restricted to %s; only %s and above can update it.", name, al, al)
		}

		command.Message = text
		command.Editor = s.User

		if forceLevel {
			command.AccessLevel = level.PGEnum()
		}

		if err := command.Update(ctx, s.Tx, boil.Whitelist(models.SimpleCommandColumns.UpdatedAt, models.SimpleCommandColumns.Message, models.SimpleCommandColumns.Editor)); err != nil {
			return err
		}

		al := flect.Pluralize(command.AccessLevel)
		return s.Replyf("Command '%s' updated, restricted to %s and above.%s", name, al, warning)
	}

	command = &models.SimpleCommand{
		Name:        name,
		ChannelID:   s.Channel.ID,
		Message:     text,
		AccessLevel: level.PGEnum(),
		Editor:      s.User,
	}

	if err := command.Insert(ctx, s.Tx, boil.Infer()); err != nil {
		return err
	}

	al := flect.Pluralize(command.AccessLevel)
	return s.Replyf("Command '%s' added, restricted to %s and above.%s", name, al, warning)
}

func cmdSimpleCommandDelete(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("<name>")
	}

	name, _ := splitSpace(args)

	if name == "" {
		return usage()
	}

	name = strings.ToLower(name)

	command, err := models.SimpleCommands(
		models.SimpleCommandWhere.ChannelID.EQ(s.Channel.ID),
		models.SimpleCommandWhere.Name.EQ(name),
		qm.For("UPDATE"),
		qm.Load(models.SimpleCommandRels.RepeatedCommand),
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return s.Replyf("Command '%s' does not exist.", name)
	}

	if err != nil {
		return err
	}

	level := newAccessLevel(command.AccessLevel)
	if !s.UserLevel.CanAccess(level) {
		return s.Replyf("Your level is %s; you cannot delete a command with level %s.", s.UserLevel.PGEnum(), command.AccessLevel)
	}

	deletedRepeat := false

	if command.R.RepeatedCommand != nil {
		s.Deps.UpdateRepeat(command.R.RepeatedCommand.ID, false, 0, 0)

		err = command.R.RepeatedCommand.Delete(ctx, s.Tx)
		if err != nil {
			return err
		}

		deletedRepeat = true
	}

	err = command.Delete(ctx, s.Tx)
	if err != nil {
		return err
	}

	if deletedRepeat {
		return s.Replyf("Command '%s' and its repeat have been deleted.", name)
	}

	return s.Replyf("Command '%s' deleted.", name)
}

func cmdSimpleCommandRestrict(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("<name> everyone|regulars|subs|mods|broadcaster|admin")
	}

	name, level := splitSpace(args)

	if name == "" {
		return usage()
	}

	command, err := models.SimpleCommands(
		models.SimpleCommandWhere.ChannelID.EQ(s.Channel.ID),
		models.SimpleCommandWhere.Name.EQ(name),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return s.Replyf("Command '%s' does not exist.", name)
	}

	if err != nil {
		return err
	}

	if level == "" {
		return s.Replyf("Command '%s' is restricted to %s and above.", name, flect.Pluralize(command.AccessLevel))
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

	if !s.UserLevel.CanAccess(newAccessLevel(command.AccessLevel)) {
		return s.Replyf("Your level is %s; you cannot restrict a command with level %s.", s.UserLevel.PGEnum(), command.AccessLevel)
	}

	if !s.UserLevel.CanAccess(newAccessLevel(newLevel)) {
		return s.Replyf("Your level is %s; you cannot restrict a command to level %s.", s.UserLevel.PGEnum(), newLevel)
	}

	command.AccessLevel = newLevel
	command.Editor = s.User

	if err := command.Update(ctx, s.Tx, boil.Whitelist(models.SimpleCommandColumns.UpdatedAt, models.SimpleCommandColumns.AccessLevel, models.SimpleCommandColumns.Editor)); err != nil {
		return err
	}

	return s.Replyf("Command '%s' restricted to %s and above.", name, flect.Pluralize(command.AccessLevel))
}

func cmdSimpleCommandProperty(ctx context.Context, s *session, prop string, args string) error {
	name, _ := splitSpace(args)

	if name == "" {
		return s.ReplyUsage("<name>")
	}

	command, err := models.SimpleCommands(
		models.SimpleCommandWhere.ChannelID.EQ(s.Channel.ID),
		models.SimpleCommandWhere.Name.EQ(name),
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return s.Replyf("Command '%s' does not exist.", name)
	}

	if err != nil {
		return err
	}

	switch prop {
	case "editor", "author":
		return s.Replyf("Command '%s' was last modified by %s.", name, command.Editor) // TODO: include the date/time?
	case "count":
		u := "times"

		if command.Count == 1 {
			u = "time"
		}

		return s.Replyf("Command '%s' has been used %d %s.", name, command.Count, u)
	}

	panic("unreachable")
}

func cmdSimpleCommandRename(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("<old> <new>")
	}

	oldName, args := splitSpace(args)
	newName, _ := splitSpace(args)

	if oldName == "" || newName == "" {
		return usage()
	}

	oldName = strings.ToLower(oldName)
	newName = strings.ToLower(newName)

	if oldName == newName {
		return s.Replyf("'%s' is already called '%s'!", oldName, oldName)
	}

	command, err := models.SimpleCommands(
		models.SimpleCommandWhere.ChannelID.EQ(s.Channel.ID),
		models.SimpleCommandWhere.Name.EQ(oldName),
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return s.Replyf("Command '%s' does not exist.", oldName)
	}

	if err != nil {
		return err
	}

	level := newAccessLevel(command.AccessLevel)
	if !s.UserLevel.CanAccess(level) {
		return s.Replyf("Your level is %s; you cannot rename a command with level %s.", s.UserLevel.PGEnum(), command.AccessLevel)
	}

	exists, err := models.SimpleCommands(
		models.SimpleCommandWhere.ChannelID.EQ(s.Channel.ID),
		models.SimpleCommandWhere.Name.EQ(newName),
	).Exists(ctx, s.Tx)

	if err != nil {
		return err
	}

	if exists {
		return s.Replyf("Command '%s' already exists.", newName)
	}

	command.Name = newName

	if err := command.Update(ctx, s.Tx, boil.Whitelist(models.SimpleCommandColumns.UpdatedAt, models.SimpleCommandColumns.Name)); err != nil {
		return err
	}

	return s.Replyf("Command '%s' has been renamed to '%s'.", oldName, newName)
}

func cmdSimpleCommandGet(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("<name>")
	}

	name, _ := splitSpace(args)

	if name == "" {
		return usage()
	}

	name = strings.ToLower(name)

	command, err := models.SimpleCommands(
		models.SimpleCommandWhere.ChannelID.EQ(s.Channel.ID),
		models.SimpleCommandWhere.Name.EQ(name),
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return s.Replyf("Command '%s' does not exist.", name)
	}

	if err != nil {
		return err
	}

	return s.Replyf("Command '%s': %s", name, command.Message)
}

func init() {
	flect.AddPlural("everyone", "everyone")
}
