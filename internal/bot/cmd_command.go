package bot

import (
	"context"
	"database/sql"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

var ccCommands = newHandlerMap(map[string]handlerFunc{
	"add":             {fn: cmdCommandAddNormal, minLevel: levelModerator},
	"addb":            {fn: cmdCommandAddBroadcaster, minLevel: levelModerator},
	"addbroadcaster":  {fn: cmdCommandAddBroadcaster, minLevel: levelModerator},
	"addbroadcasters": {fn: cmdCommandAddBroadcaster, minLevel: levelModerator},
	"addo":            {fn: cmdCommandAddBroadcaster, minLevel: levelModerator},
	"addowner":        {fn: cmdCommandAddBroadcaster, minLevel: levelModerator},
	"addowners":       {fn: cmdCommandAddBroadcaster, minLevel: levelModerator},
	"addstreamer":     {fn: cmdCommandAddBroadcaster, minLevel: levelModerator},
	"addstreamers":    {fn: cmdCommandAddBroadcaster, minLevel: levelModerator},
	"addm":            {fn: cmdCommandAddModerator, minLevel: levelModerator},
	"addmod":          {fn: cmdCommandAddModerator, minLevel: levelModerator},
	"addmods":         {fn: cmdCommandAddModerator, minLevel: levelModerator},
	"adds":            {fn: cmdCommandAddSubscriber, minLevel: levelModerator},
	"addsub":          {fn: cmdCommandAddSubscriber, minLevel: levelModerator},
	"addsubs":         {fn: cmdCommandAddSubscriber, minLevel: levelModerator},
	"adde":            {fn: cmdCommandAddEveryone, minLevel: levelModerator},
	"adda":            {fn: cmdCommandAddEveryone, minLevel: levelModerator},
	"addeveryone":     {fn: cmdCommandAddEveryone, minLevel: levelModerator},
	"addall":          {fn: cmdCommandAddEveryone, minLevel: levelModerator},
	"delete":          {fn: cmdCommandDelete, minLevel: levelModerator},
	"remove":          {fn: cmdCommandDelete, minLevel: levelModerator},
	"restrict":        {fn: cmdCommandRestrict, minLevel: levelModerator},
	"editor":          {fn: cmdCommandProperty, minLevel: levelModerator},
	"author":          {fn: cmdCommandProperty, minLevel: levelModerator},
	"count":           {fn: cmdCommandProperty, minLevel: levelModerator},
	"rename":          {fn: cmdCommandRename, minLevel: levelModerator},
	"get":             {fn: cmdCommandGet, minLevel: levelModerator},
	// TODO: clone
})

func cmdCommand(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	ok, err := ccCommands.run(ctx, s, subcommand, args)
	if err != nil {
		return err
	}

	if !ok {
		return s.ReplyUsage("add|delete|restrict|...")
	}

	return nil
}

func cmdCommandAddNormal(ctx context.Context, s *session, cmd string, args string) error {
	return cmdCommandAdd(ctx, s, args, levelSubscriber, false)
}

func cmdCommandAddBroadcaster(ctx context.Context, s *session, cmd string, args string) error {
	return cmdCommandAdd(ctx, s, args, levelBroadcaster, true)
}

func cmdCommandAddModerator(ctx context.Context, s *session, cmd string, args string) error {
	return cmdCommandAdd(ctx, s, args, levelModerator, true)
}

func cmdCommandAddSubscriber(ctx context.Context, s *session, cmd string, args string) error {
	return cmdCommandAdd(ctx, s, args, levelSubscriber, true)
}

func cmdCommandAddEveryone(ctx context.Context, s *session, cmd string, args string) error {
	return cmdCommandAdd(ctx, s, args, levelEveryone, true)
}

func cmdCommandAdd(ctx context.Context, s *session, args string, level accessLevel, forceLevel bool) error {
	usage := func() error {
		return s.ReplyUsage("<name> <text>")
	}

	name, text := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" || text == "" {
		return usage()
	}

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

	info, command, err := findCustomCommand(ctx, s, name, true)
	if err != nil {
		return err
	}

	if info != nil && command == nil {
		return s.Replyf("A command or list with name '%s' already exists.", name)
	}

	update := command != nil

	if !s.UserLevel.CanAccess(level) {
		a := "add"
		if update {
			a = "update"
		}

		return s.Replyf("Your level is %s; you cannot %s a command with level %s.", s.UserLevel.PGEnum(), a, level.PGEnum())
	}

	if update {
		if !s.UserLevel.CanAccessPG(info.AccessLevel) {
			al := flect.Pluralize(info.AccessLevel)
			return s.Replyf("Command '%s' is restricted to %s; only %s and above can update it.", name, al, al)
		}

		command.Message = text

		if err := command.Update(ctx, s.Tx, boil.Whitelist(models.CustomCommandColumns.UpdatedAt, models.CustomCommandColumns.Message)); err != nil {
			return err
		}

		info.Editor = s.User

		if forceLevel {
			info.AccessLevel = level.PGEnum()
		}

		if err := info.Update(ctx, s.Tx, boil.Whitelist(models.CommandInfoColumns.UpdatedAt, models.CommandInfoColumns.AccessLevel, models.CommandInfoColumns.Editor)); err != nil {
			return err
		}

		al := flect.Pluralize(info.AccessLevel)
		return s.Replyf("Command '%s' updated, restricted to %s and above.%s", name, al, warning)
	}

	command = &models.CustomCommand{
		ChannelID: s.Channel.ID,
		Message:   text,
	}

	if err := command.Insert(ctx, s.Tx, boil.Infer()); err != nil {
		return err
	}

	info = &models.CommandInfo{
		ChannelID:       s.Channel.ID,
		Name:            name,
		CustomCommandID: null.Int64From(command.ID),
		AccessLevel:     level.PGEnum(),
		Creator:         s.User,
		Editor:          s.User,
	}

	if err := info.Insert(ctx, s.Tx, boil.Infer()); err != nil {
		return err
	}

	al := flect.Pluralize(info.AccessLevel)
	return s.Replyf("Command '%s' added, restricted to %s and above.%s", name, al, warning)
}

func cmdCommandDelete(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("<name>")
	}

	name, _ := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return usage()
	}

	info, command, err := findCustomCommand(ctx, s, name, true)
	if err != nil {
		return err
	}

	if info == nil {
		return s.Replyf("Command '%s' does not exist.", name)
	}

	if command == nil {
		return s.Replyf("'%s' is not a custom command.", name)
	}

	if !s.UserLevel.CanAccessPG(info.AccessLevel) {
		return s.Replyf("Your level is %s; you cannot delete a command with level %s.", s.UserLevel.PGEnum(), info.AccessLevel)
	}

	repeated, scheduled, err := modelsx.DeleteCommandInfo(ctx, s.Tx, info)
	if err != nil {
		return err
	}

	deletedRepeat := false

	if repeated != nil {
		deletedRepeat = true
		s.Deps.UpdateRepeat(repeated.ID, false, 0, 0)
	}

	if scheduled != nil {
		deletedRepeat = true
		s.Deps.UpdateSchedule(scheduled.ID, false, nil)
	}

	if deletedRepeat {
		return s.Replyf("Command '%s' and its repeat/schedule have been deleted.", name)
	}

	return s.Replyf("Command '%s' deleted.", name)
}

func cmdCommandRestrict(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("<name> everyone|regulars|subs|mods|broadcaster|admin")
	}

	name, level := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return usage()
	}

	info, err := s.Channel.CommandInfos(models.CommandInfoWhere.Name.EQ(name), qm.For("UPDATE")).One(ctx, s.Tx)
	if err != nil {
		if err == sql.ErrNoRows {
			return s.Replyf("Command '%s' does not exist.", name)
		}
		return err
	}

	if !info.CustomCommandID.Valid {
		return s.Replyf("'%s' is not a custom command.", name)
	}

	if level == "" {
		return s.Replyf("Command '%s' is restricted to %s and above.", name, flect.Pluralize(info.AccessLevel))
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

	if !s.UserLevel.CanAccessPG(info.AccessLevel) {
		return s.Replyf("Your level is %s; you cannot restrict a command with level %s.", s.UserLevel.PGEnum(), info.AccessLevel)
	}

	if !s.UserLevel.CanAccess(newAccessLevel(newLevel)) {
		return s.Replyf("Your level is %s; you cannot restrict a command to level %s.", s.UserLevel.PGEnum(), newLevel)
	}

	info.AccessLevel = newLevel
	info.Editor = s.User

	if err := info.Update(ctx, s.Tx, boil.Whitelist(models.CommandInfoColumns.UpdatedAt, models.CommandInfoColumns.AccessLevel, models.CommandInfoColumns.Editor)); err != nil {
		return err
	}

	return s.Replyf("Command '%s' restricted to %s and above.", name, flect.Pluralize(info.AccessLevel))
}

func cmdCommandProperty(ctx context.Context, s *session, prop string, args string) error {
	name, _ := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return s.ReplyUsage("<name>")
	}

	info, err := s.Channel.CommandInfos(models.CommandInfoWhere.Name.EQ(name)).One(ctx, s.Tx)
	if err != nil {
		if err == sql.ErrNoRows {
			return s.Replyf("Command '%s' does not exist.", name)
		}
		return err
	}

	if !info.CustomCommandID.Valid {
		return s.Replyf("'%s' is not a custom command.", name)
	}

	switch prop {
	case "editor", "author":
		return s.Replyf("Command '%s' was last modified by %s.", name, info.Editor) // TODO: include the date/time?
	case "count":
		u := "times"

		if info.Count == 1 {
			u = "time"
		}

		return s.Replyf("Command '%s' has been used %d %s.", name, info.Count, u)
	}

	panic("unreachable")
}

func cmdCommandRename(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("<old> <new>")
	}

	oldName, args := splitSpace(args)
	newName, _ := splitSpace(args)

	oldName = cleanCommandName(oldName)
	newName = cleanCommandName(newName)

	if oldName == "" || newName == "" {
		return usage()
	}

	if oldName == newName {
		return s.Replyf("'%s' is already called '%s'!", oldName, oldName)
	}

	info, err := s.Channel.CommandInfos(models.CommandInfoWhere.Name.EQ(oldName), qm.For("UPDATE")).One(ctx, s.Tx)
	if err != nil {
		if err == sql.ErrNoRows {
			return s.Replyf("Command '%s' does not exist.", oldName)
		}
		return err
	}

	if !info.CustomCommandID.Valid {
		return s.Replyf("'%s' is not a custom command.", oldName)
	}

	level := newAccessLevel(info.AccessLevel)
	if !s.UserLevel.CanAccess(level) {
		return s.Replyf("Your level is %s; you cannot rename a command with level %s.", s.UserLevel.PGEnum(), info.AccessLevel)
	}

	exists, err := s.Channel.CommandInfos(models.CommandInfoWhere.Name.EQ(newName)).Exists(ctx, s.Tx)
	if err != nil {
		return err
	}

	if exists {
		return s.Replyf("A command or list with name '%s' already exists.", newName)
	}

	info.Name = newName
	info.Editor = s.User

	if err := info.Update(ctx, s.Tx, boil.Whitelist(models.CommandInfoColumns.UpdatedAt, models.CommandInfoColumns.Name, models.CommandInfoColumns.Editor)); err != nil {
		return err
	}

	return s.Replyf("Command '%s' has been renamed to '%s'.", oldName, newName)
}

func cmdCommandGet(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("<name>")
	}

	name, _ := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return usage()
	}

	info, command, err := findCustomCommand(ctx, s, name, false)
	if err != nil {
		return err
	}

	if info == nil {
		return s.Replyf("Command '%s' does not exist.", name)
	}

	if command == nil {
		return s.Replyf("'%s' is not a custom command.", name)
	}

	return s.Replyf("Command '%s': %s", name, command.Message)
}

func findCustomCommand(ctx context.Context, s *session, name string, forUpdate bool) (*models.CommandInfo, *models.CustomCommand, error) {
	var mods []qm.QueryMod

	if forUpdate {
		mods = []qm.QueryMod{
			models.CommandInfoWhere.Name.EQ(name),
			qm.Load(models.CommandInfoRels.CustomCommand, qm.For("UPDATE")),
			qm.For("UPDATE"),
		}
	} else {
		mods = []qm.QueryMod{
			models.CommandInfoWhere.Name.EQ(name),
			qm.Load(models.CommandInfoRels.CustomCommand),
		}
	}

	info, err := s.Channel.CommandInfos(mods...).One(ctx, s.Tx)

	switch err {
	case nil:
		return info, info.R.CustomCommand, nil
	case sql.ErrNoRows:
		return nil, nil, nil
	default:
		return nil, nil, err
	}
}

func init() {
	flect.AddPlural("everyone", "everyone")
}
