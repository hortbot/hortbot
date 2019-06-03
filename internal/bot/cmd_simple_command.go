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

var scCommands builtinMap = map[string]builtinCommand{
	"add":             {fn: cmdSimpleCommandAddFunc(LevelSubscriber, false), minLevel: LevelModerator},
	"addb":            {fn: cmdSimpleCommandAddFunc(LevelBroadcaster, true), minLevel: LevelModerator},
	"addbroadcaster":  {fn: cmdSimpleCommandAddFunc(LevelBroadcaster, true), minLevel: LevelModerator},
	"addbroadcasters": {fn: cmdSimpleCommandAddFunc(LevelBroadcaster, true), minLevel: LevelModerator},
	"addo":            {fn: cmdSimpleCommandAddFunc(LevelBroadcaster, true), minLevel: LevelModerator},
	"addowner":        {fn: cmdSimpleCommandAddFunc(LevelBroadcaster, true), minLevel: LevelModerator},
	"addowners":       {fn: cmdSimpleCommandAddFunc(LevelBroadcaster, true), minLevel: LevelModerator},
	"addstreamer":     {fn: cmdSimpleCommandAddFunc(LevelBroadcaster, true), minLevel: LevelModerator},
	"addstreamers":    {fn: cmdSimpleCommandAddFunc(LevelBroadcaster, true), minLevel: LevelModerator},
	"addm":            {fn: cmdSimpleCommandAddFunc(LevelModerator, true), minLevel: LevelModerator},
	"addmod":          {fn: cmdSimpleCommandAddFunc(LevelModerator, true), minLevel: LevelModerator},
	"addmods":         {fn: cmdSimpleCommandAddFunc(LevelModerator, true), minLevel: LevelModerator},
	"adds":            {fn: cmdSimpleCommandAddFunc(LevelSubscriber, true), minLevel: LevelModerator},
	"addsub":          {fn: cmdSimpleCommandAddFunc(LevelSubscriber, true), minLevel: LevelModerator},
	"addsubs":         {fn: cmdSimpleCommandAddFunc(LevelSubscriber, true), minLevel: LevelModerator},
	"adde":            {fn: cmdSimpleCommandAddFunc(LevelEveryone, true), minLevel: LevelModerator},
	"adda":            {fn: cmdSimpleCommandAddFunc(LevelEveryone, true), minLevel: LevelModerator},
	"addeveryone":     {fn: cmdSimpleCommandAddFunc(LevelEveryone, true), minLevel: LevelModerator},
	"addall":          {fn: cmdSimpleCommandAddFunc(LevelEveryone, true), minLevel: LevelModerator},
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

func cmdSimpleCommandAddFunc(level AccessLevel, forceLevel bool) func(ctx context.Context, s *Session, cmd string, args string) error {
	return func(ctx context.Context, s *Session, cmd string, args string) error {
		return cmdSimpleCommandAdd(ctx, s, args, level, forceLevel)
	}
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
		return s.Replyf("command name %s is reserved", name)
	}

	// TODO: remove this warning
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

		return s.Replyf("your level is %s; you cannot %s a command with level %s", s.UserLevel.PGEnum(), a, level.PGEnum())
	}

	if update {
		if !s.UserLevel.CanAccess(NewAccessLevel(command.AccessLevel)) {
			al := flect.Pluralize(command.AccessLevel)
			return s.Replyf("command %s is restricted to %s; only %s and above can update it", name, al, al)
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
		return s.Replyf("command %s updated, restricted to %s and above%s", name, al, warning)
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
	return s.Replyf("command %s added, restricted to %s and above%s", name, al, warning)
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
		return s.Replyf("command %s does not exist", name)
	}

	if err != nil {
		return err
	}

	level := NewAccessLevel(command.AccessLevel)
	if !s.UserLevel.CanAccess(level) {
		return s.Replyf("your level is %s; you cannot delete a command with level %s", s.UserLevel.PGEnum(), command.AccessLevel)
	}

	err = command.Delete(ctx, s.Tx)
	if err != nil {
		return err
	}

	return s.Replyf("command %s deleted", name)
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
		return s.Replyf("command %s does not exist", name)
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

	if !s.UserLevel.CanAccess(NewAccessLevel(command.AccessLevel)) {
		return s.Replyf("your level is %s; you cannot restrict a command with level %s", s.UserLevel.PGEnum(), command.AccessLevel)
	}

	if !s.UserLevel.CanAccess(NewAccessLevel(newLevel)) {
		return s.Replyf("your level is %s; you cannot restrict a command to level %s", s.UserLevel.PGEnum(), newLevel)
	}

	command.AccessLevel = newLevel
	command.Editor = s.User

	if err := command.Update(ctx, s.Tx, boil.Whitelist(models.SimpleCommandColumns.UpdatedAt, models.SimpleCommandColumns.AccessLevel, models.SimpleCommandColumns.Editor)); err != nil {
		return err
	}

	return s.Replyf("command %s restricted to %s and above", name, flect.Pluralize(command.AccessLevel))
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
		return s.Replyf("command %s does not exist", name)
	}

	if err != nil {
		return err
	}

	switch prop {
	case "editor", "author":
		return s.Replyf("command %s was last modified by %s", name, command.Editor) // TODO: include the date/time?
	case "count":
		u := "times"

		if command.Count == 1 {
			u = "time"
		}

		return s.Replyf("command %s has been used %d %s", name, command.Count, u)
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
		return s.Replyf("%s is already called %s!", oldName, oldName)
	}

	command, err := models.SimpleCommands(
		models.SimpleCommandWhere.ChannelID.EQ(s.Channel.ID),
		models.SimpleCommandWhere.Name.EQ(oldName),
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return s.Replyf("command %s does not exist", oldName)
	}

	if err != nil {
		return err
	}

	level := NewAccessLevel(command.AccessLevel)
	if !s.UserLevel.CanAccess(level) {
		return s.Replyf("your level is %s; you cannot rename a command with level %s", s.UserLevel.PGEnum(), command.AccessLevel)
	}

	exists, err := models.SimpleCommands(
		models.SimpleCommandWhere.ChannelID.EQ(s.Channel.ID),
		models.SimpleCommandWhere.Name.EQ(newName),
	).Exists(ctx, s.Tx)

	if err != nil {
		return err
	}

	if exists {
		return s.Replyf("command %s already exists", newName)
	}

	command.Name = newName

	if err := command.Update(ctx, s.Tx, boil.Whitelist(models.SimpleCommandColumns.UpdatedAt, models.SimpleCommandColumns.Name)); err != nil {
		return err
	}

	return s.Replyf("command %s has been renamed to %s", oldName, newName)
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
		return s.Replyf("command %s does not exist", name)
	}

	if err != nil {
		return err
	}

	return s.Replyf("command %s: %s", name, command.Message)
}

func init() {
	flect.AddPlural("everyone", "everyone")
}
