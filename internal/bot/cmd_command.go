package bot

import (
	"context"
	"database/sql"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

var ccCommands = newHandlerMap(map[string]handlerFunc{
	"add":             {fn: cmdCommandAddNormal, minLevel: levelModerator},
	"edit":            {fn: cmdCommandAddNormal, minLevel: levelModerator},
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
	"rm":              {fn: cmdCommandDelete, minLevel: levelModerator},
	"restrict":        {fn: cmdCommandRestrict, minLevel: levelModerator},
	"editor":          {fn: cmdCommandProperty, minLevel: levelModerator},
	"author":          {fn: cmdCommandProperty, minLevel: levelModerator},
	"count":           {fn: cmdCommandProperty, minLevel: levelModerator},
	"rename":          {fn: cmdCommandRename, minLevel: levelModerator},
	"get":             {fn: cmdCommandGet, minLevel: levelModerator},
	"clone":           {fn: cmdCommandClone, minLevel: levelModerator},
	"exec":            {fn: cmdCommandExec, minLevel: levelModerator},
})

func cmdCommand(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	ok, err := ccCommands.Run(ctx, s, subcommand, args)
	if err != nil {
		return err
	}

	if !ok {
		return s.ReplyUsage(ctx, "add|delete|restrict|...")
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
		return s.ReplyUsage(ctx, "<name> <text>")
	}

	name, text := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" || text == "" {
		return usage()
	}

	if reservedCommandNames[name] {
		return s.Replyf(ctx, "Command name '%s' is reserved.", name)
	}

	var warning string
	if isBuiltinName(name) {
		warning = " Warning: '" + name + "' is a builtin command and will now only be accessible via " + s.Channel.Prefix + "builtin " + name + "."
	} else if prefixAndName, ok := isModerationCommand(s.Channel.Prefix, name); ok {
		warning = " Warning: '" + prefixAndName + "' is a moderation command; your custom command may not work."
	}

	if _, malformed := cbp.Parse(text); malformed {
		warning += " Warning: command contains stray (_ or _) separators and may not be processed correctly."
	}

	info, command, err := findCustomCommand(ctx, s, name, true)
	if err != nil {
		return err
	}

	if info != nil && command == nil {
		return s.Replyf(ctx, "A command or list with name '%s' already exists.", name)
	}

	update := command != nil

	if !s.UserLevel.CanAccess(level) {
		a := "add"
		if update {
			a = "update"
		}

		return s.Replyf(ctx, "Your level is %s; you cannot %s a command with level %s.", s.UserLevel.PGEnum(), a, level.PGEnum())
	}

	if !forceLevel {
		orig := level
		level = levelModerator

		switch {
		case strings.Contains(text, "(_PURGE_)"):
		case strings.Contains(text, "(_TIMEOUT_)"):
		case strings.Contains(text, "(_BAN_)"):
		case strings.Contains(text, "(_VARS_") && (strings.Contains(text, "_INCREMENT_") || strings.Contains(text, "_DECREMENT_") || strings.Contains(text, "_SET_")):
		case strings.Contains(text, "(_SUBMODE_ON_)"):
		case strings.Contains(text, "(_SUBMODE_OFF_)"):
		default:
			level = orig
		}
	}

	if update {
		if !s.UserLevel.CanAccessPG(info.AccessLevel) {
			al := flect.Pluralize(info.AccessLevel)
			return s.Replyf(ctx, "Command '%s' is restricted to %s; only %s and above can update it.", name, al, al)
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
		return s.Replyf(ctx, "Command '%s' updated, restricted to %s and above.%s", name, al, warning)
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
	return s.Replyf(ctx, "Command '%s' added, restricted to %s and above.%s", name, al, warning)
}

func cmdCommandDelete(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<name>")
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
		return s.Replyf(ctx, "Command '%s' does not exist.", name)
	}

	if command == nil {
		return s.Replyf(ctx, "'%s' is not a custom command.", name)
	}

	if !s.UserLevel.CanAccessPG(info.AccessLevel) {
		return s.Replyf(ctx, "Your level is %s; you cannot delete a command with level %s.", s.UserLevel.PGEnum(), info.AccessLevel)
	}

	repeated, scheduled, err := modelsx.DeleteCommandInfo(ctx, s.Tx, info)
	if err != nil {
		return err
	}

	deletedRepeat := false

	if repeated != nil {
		deletedRepeat = true
		if err := s.Deps.RemoveRepeat(ctx, repeated.ID); err != nil {
			return err
		}
	}

	if scheduled != nil {
		deletedRepeat = true
		if err := s.Deps.RemoveScheduled(ctx, scheduled.ID); err != nil {
			return err
		}
	}

	if deletedRepeat {
		return s.Replyf(ctx, "Command '%s' and its repeat/schedule have been deleted.", name)
	}

	return s.Replyf(ctx, "Command '%s' deleted.", name)
}

func cmdCommandRestrict(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<name> everyone|regulars|subs|mods|broadcaster|admin")
	}

	name, level := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return usage()
	}

	info, err := s.Channel.CommandInfos(models.CommandInfoWhere.Name.EQ(name), qm.For("UPDATE")).One(ctx, s.Tx)
	if err != nil {
		if err == sql.ErrNoRows {
			return s.Replyf(ctx, "Command '%s' does not exist.", name)
		}
		return err
	}

	if !info.CustomCommandID.Valid {
		return s.Replyf(ctx, "'%s' is not a custom command.", name)
	}

	if level == "" {
		return s.Replyf(ctx, "Command '%s' is restricted to %s and above.", name, flect.Pluralize(info.AccessLevel))
	}

	level = strings.ToLower(level)

	newLevel := parseLevelPG(level)
	if newLevel == "" {
		return usage()
	}

	if !s.UserLevel.CanAccessPG(info.AccessLevel) {
		return s.Replyf(ctx, "Your level is %s; you cannot restrict a command with level %s.", s.UserLevel.PGEnum(), info.AccessLevel)
	}

	if !s.UserLevel.CanAccess(newAccessLevel(newLevel)) {
		return s.Replyf(ctx, "Your level is %s; you cannot restrict a command to level %s.", s.UserLevel.PGEnum(), newLevel)
	}

	info.AccessLevel = newLevel
	info.Editor = s.User

	if err := info.Update(ctx, s.Tx, boil.Whitelist(models.CommandInfoColumns.UpdatedAt, models.CommandInfoColumns.AccessLevel, models.CommandInfoColumns.Editor)); err != nil {
		return err
	}

	return s.Replyf(ctx, "Command '%s' restricted to %s and above.", name, flect.Pluralize(info.AccessLevel))
}

func cmdCommandProperty(ctx context.Context, s *session, prop string, args string) error {
	name, _ := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return s.ReplyUsage(ctx, "<name>")
	}

	info, err := s.Channel.CommandInfos(models.CommandInfoWhere.Name.EQ(name)).One(ctx, s.Tx)
	if err != nil {
		if err == sql.ErrNoRows {
			return s.Replyf(ctx, "Command '%s' does not exist.", name)
		}
		return err
	}

	if !info.CustomCommandID.Valid {
		return s.Replyf(ctx, "'%s' is not a custom command.", name)
	}

	switch prop {
	case "editor", "author":
		return s.Replyf(ctx, "Command '%s' was last modified by %s.", name, info.Editor)
	case "count":
		u := "times"

		if info.Count == 1 {
			u = "time"
		}

		return s.Replyf(ctx, "Command '%s' has been used %d %s.", name, info.Count, u)
	}

	panic("unreachable")
}

func cmdCommandRename(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<old> <new>")
	}

	oldName, args := splitSpace(args)
	newName, _ := splitSpace(args)

	oldName = cleanCommandName(oldName)
	newName = cleanCommandName(newName)

	if oldName == "" || newName == "" {
		return usage()
	}

	if oldName == newName {
		return s.Replyf(ctx, "'%s' is already called '%s'!", oldName, oldName)
	}

	info, err := s.Channel.CommandInfos(models.CommandInfoWhere.Name.EQ(oldName), qm.For("UPDATE")).One(ctx, s.Tx)
	if err != nil {
		if err == sql.ErrNoRows {
			return s.Replyf(ctx, "Command '%s' does not exist.", oldName)
		}
		return err
	}

	if !info.CustomCommandID.Valid {
		return s.Replyf(ctx, "'%s' is not a custom command.", oldName)
	}

	level := newAccessLevel(info.AccessLevel)
	if !s.UserLevel.CanAccess(level) {
		return s.Replyf(ctx, "Your level is %s; you cannot rename a command with level %s.", s.UserLevel.PGEnum(), info.AccessLevel)
	}

	exists, err := s.Channel.CommandInfos(models.CommandInfoWhere.Name.EQ(newName)).Exists(ctx, s.Tx)
	if err != nil {
		return err
	}

	if exists {
		return s.Replyf(ctx, "A command or list with name '%s' already exists.", newName)
	}

	info.Name = newName
	info.Editor = s.User

	if err := info.Update(ctx, s.Tx, boil.Whitelist(models.CommandInfoColumns.UpdatedAt, models.CommandInfoColumns.Name, models.CommandInfoColumns.Editor)); err != nil {
		return err
	}

	return s.Replyf(ctx, "Command '%s' has been renamed to '%s'.", oldName, newName)
}

func cmdCommandGet(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<name>")
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
		return s.Replyf(ctx, "Command '%s' does not exist.", name)
	}

	if command == nil {
		return s.Replyf(ctx, "'%s' is not a custom command.", name)
	}

	return s.Replyf(ctx, "Command '%s': %s", name, command.Message)
}

func cmdCommandClone(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "#<channel> <name>")
	}

	other, args := splitSpace(args)
	name, _ := splitSpace(args)
	name = cleanCommandName(name)

	if len(other) < 2 || name == "" {
		return usage()
	}

	if other[0] != '#' {
		return usage()
	}

	other = strings.ToLower(other[1:])

	exists, err := s.Channel.CommandInfos(models.CommandInfoWhere.Name.EQ(name)).Exists(ctx, s.Tx)
	if err != nil {
		return err
	}
	if exists {
		return s.Replyf(ctx, "A command or list with name '%s' already exists.", name)
	}

	otherChannel, err := models.Channels(models.ChannelWhere.Name.EQ(other)).One(ctx, s.Tx)
	if err != nil {
		if err == sql.ErrNoRows {
			return s.Replyf(ctx, "Channel %s does not exist.", other)
		}
		return err
	}

	oldInfo, commandMessage, found, err := modelsx.FindCommand(ctx, s.Tx, otherChannel.ID, name, false)
	if err != nil {
		return err
	}

	if !found {
		return s.Replyf(ctx, "Channel %s does not have a command named '%s'.", other, name)
	}

	if !commandMessage.Valid {
		return s.Replyf(ctx, "'%s' is not a command.", name)
	}

	command := &models.CustomCommand{
		ChannelID: s.Channel.ID,
		Message:   commandMessage.String,
	}

	if err := command.Insert(ctx, s.Tx, boil.Infer()); err != nil {
		return err
	}

	info := &models.CommandInfo{
		ChannelID:       s.Channel.ID,
		Name:            name,
		CustomCommandID: null.Int64From(command.ID),
		AccessLevel:     oldInfo.AccessLevel,
		Creator:         s.User,
		Editor:          s.User,
	}

	if err := info.Insert(ctx, s.Tx, boil.Infer()); err != nil {
		return err
	}

	return s.Replyf(ctx, "Command '%s' has been cloned from channel %s.", name, other)
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

func cmdCommandExec(ctx context.Context, s *session, _ string, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "<command string>")
	}

	reply, err := processCommand(ctx, s, args)
	if err != nil {
		return err
	}

	return s.Reply(ctx, reply)
}
