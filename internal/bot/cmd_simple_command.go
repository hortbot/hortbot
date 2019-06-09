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
	"add":             {fn: cmdSimpleCommandAddNormal, minLevel: LevelModerator},
	"addb":            {fn: cmdSimpleCommandAddBroadcaster, minLevel: LevelModerator},
	"addbroadcaster":  {fn: cmdSimpleCommandAddBroadcaster, minLevel: LevelModerator},
	"addbroadcasters": {fn: cmdSimpleCommandAddBroadcaster, minLevel: LevelModerator},
	"addo":            {fn: cmdSimpleCommandAddBroadcaster, minLevel: LevelModerator},
	"addowner":        {fn: cmdSimpleCommandAddBroadcaster, minLevel: LevelModerator},
	"addowners":       {fn: cmdSimpleCommandAddBroadcaster, minLevel: LevelModerator},
	"addstreamer":     {fn: cmdSimpleCommandAddBroadcaster, minLevel: LevelModerator},
	"addstreamers":    {fn: cmdSimpleCommandAddBroadcaster, minLevel: LevelModerator},
	"addm":            {fn: cmdSimpleCommandAddModerator, minLevel: LevelModerator},
	"addmod":          {fn: cmdSimpleCommandAddModerator, minLevel: LevelModerator},
	"addmods":         {fn: cmdSimpleCommandAddModerator, minLevel: LevelModerator},
	"adds":            {fn: cmdSimpleCommandAddSubscriber, minLevel: LevelModerator},
	"addsub":          {fn: cmdSimpleCommandAddSubscriber, minLevel: LevelModerator},
	"addsubs":         {fn: cmdSimpleCommandAddSubscriber, minLevel: LevelModerator},
	"adde":            {fn: cmdSimpleCommandAddEveryone, minLevel: LevelModerator},
	"adda":            {fn: cmdSimpleCommandAddEveryone, minLevel: LevelModerator},
	"addeveryone":     {fn: cmdSimpleCommandAddEveryone, minLevel: LevelModerator},
	"addall":          {fn: cmdSimpleCommandAddEveryone, minLevel: LevelModerator},
	"delete":          {fn: cmdSimpleCommandDelete, minLevel: LevelModerator},
	"remove":          {fn: cmdSimpleCommandDelete, minLevel: LevelModerator},
	"restrict":        {fn: cmdSimpleCommandRestrict, minLevel: LevelModerator},
	"editor":          {fn: cmdSimpleCommandProperty, minLevel: LevelModerator},
	"author":          {fn: cmdSimpleCommandProperty, minLevel: LevelModerator},
	"count":           {fn: cmdSimpleCommandProperty, minLevel: LevelModerator},
	"rename":          {fn: cmdSimpleCommandRename, minLevel: LevelModerator},
	"get":             {fn: cmdSimpleCommandGet, minLevel: LevelModerator},
	// TODO: clone
}

func cmdSimpleCommand(ctx context.Context, s *Session, cmd string, args string) error {
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

func cmdSimpleCommandAddNormal(ctx context.Context, s *Session, cmd string, args string) error {
	return cmdSimpleCommandAdd(ctx, s, args, LevelSubscriber, false)
}

func cmdSimpleCommandAddBroadcaster(ctx context.Context, s *Session, cmd string, args string) error {
	return cmdSimpleCommandAdd(ctx, s, args, LevelBroadcaster, true)
}

func cmdSimpleCommandAddModerator(ctx context.Context, s *Session, cmd string, args string) error {
	return cmdSimpleCommandAdd(ctx, s, args, LevelModerator, true)
}

func cmdSimpleCommandAddSubscriber(ctx context.Context, s *Session, cmd string, args string) error {
	return cmdSimpleCommandAdd(ctx, s, args, LevelSubscriber, true)
}

func cmdSimpleCommandAddEveryone(ctx context.Context, s *Session, cmd string, args string) error {
	return cmdSimpleCommandAdd(ctx, s, args, LevelEveryone, true)
}

func cmdSimpleCommandAdd(ctx context.Context, s *Session, args string, level AccessLevel, forceLevel bool) error {
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
		if !s.UserLevel.CanAccess(NewAccessLevel(command.AccessLevel)) {
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

func cmdSimpleCommandDelete(ctx context.Context, s *Session, cmd string, args string) error {
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
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return s.Replyf("Command '%s' does not exist.", name)
	}

	if err != nil {
		return err
	}

	level := NewAccessLevel(command.AccessLevel)
	if !s.UserLevel.CanAccess(level) {
		return s.Replyf("Your level is %s; you cannot delete a command with level %s.", s.UserLevel.PGEnum(), command.AccessLevel)
	}

	err = command.Delete(ctx, s.Tx)
	if err != nil {
		return err
	}

	return s.Replyf("Command '%s' deleted.", name)
}

func cmdSimpleCommandRestrict(ctx context.Context, s *Session, cmd string, args string) error {
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

	if !s.UserLevel.CanAccess(NewAccessLevel(command.AccessLevel)) {
		return s.Replyf("Your level is %s; you cannot restrict a command with level %s.", s.UserLevel.PGEnum(), command.AccessLevel)
	}

	if !s.UserLevel.CanAccess(NewAccessLevel(newLevel)) {
		return s.Replyf("Your level is %s; you cannot restrict a command to level %s.", s.UserLevel.PGEnum(), newLevel)
	}

	command.AccessLevel = newLevel
	command.Editor = s.User

	if err := command.Update(ctx, s.Tx, boil.Whitelist(models.SimpleCommandColumns.UpdatedAt, models.SimpleCommandColumns.AccessLevel, models.SimpleCommandColumns.Editor)); err != nil {
		return err
	}

	return s.Replyf("Command '%s' restricted to %s and above.", name, flect.Pluralize(command.AccessLevel))
}

func cmdSimpleCommandProperty(ctx context.Context, s *Session, prop string, args string) error {
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

func cmdSimpleCommandRename(ctx context.Context, s *Session, cmd string, args string) error {
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

	level := NewAccessLevel(command.AccessLevel)
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

func cmdSimpleCommandGet(ctx context.Context, s *Session, cmd string, args string) error {
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
