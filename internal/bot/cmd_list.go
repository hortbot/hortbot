package bot

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"go.opencensus.io/trace"
)

// TODO: Merge the code between custom commands and lists; they are identical other than some wordings.

var listCommands = newHandlerMap(map[string]handlerFunc{
	"add":             {fn: cmdListAddSubscriber, minLevel: AccessLevelModerator},
	"addb":            {fn: cmdListAddBroadcaster, minLevel: AccessLevelModerator},
	"addbroadcaster":  {fn: cmdListAddBroadcaster, minLevel: AccessLevelModerator},
	"addbroadcasters": {fn: cmdListAddBroadcaster, minLevel: AccessLevelModerator},
	"addo":            {fn: cmdListAddBroadcaster, minLevel: AccessLevelModerator},
	"addowner":        {fn: cmdListAddBroadcaster, minLevel: AccessLevelModerator},
	"addowners":       {fn: cmdListAddBroadcaster, minLevel: AccessLevelModerator},
	"addstreamer":     {fn: cmdListAddBroadcaster, minLevel: AccessLevelModerator},
	"addstreamers":    {fn: cmdListAddBroadcaster, minLevel: AccessLevelModerator},
	"addm":            {fn: cmdListAddModerator, minLevel: AccessLevelModerator},
	"addmod":          {fn: cmdListAddModerator, minLevel: AccessLevelModerator},
	"addmods":         {fn: cmdListAddModerator, minLevel: AccessLevelModerator},
	"adds":            {fn: cmdListAddSubscriber, minLevel: AccessLevelModerator},
	"addsub":          {fn: cmdListAddSubscriber, minLevel: AccessLevelModerator},
	"addsubs":         {fn: cmdListAddSubscriber, minLevel: AccessLevelModerator},
	"adde":            {fn: cmdListAddEveryone, minLevel: AccessLevelModerator},
	"adda":            {fn: cmdListAddEveryone, minLevel: AccessLevelModerator},
	"addeveryone":     {fn: cmdListAddEveryone, minLevel: AccessLevelModerator},
	"addall":          {fn: cmdListAddEveryone, minLevel: AccessLevelModerator},
	"delete":          {fn: cmdListDelete, minLevel: AccessLevelModerator},
	"remove":          {fn: cmdListDelete, minLevel: AccessLevelModerator},
	"rm":              {fn: cmdListDelete, minLevel: AccessLevelModerator},
	"restrict":        {fn: cmdListRestrict, minLevel: AccessLevelModerator},
	"rename":          {fn: cmdListRename, minLevel: AccessLevelModerator},
})

func cmdList(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	ok, err := listCommands.Run(ctx, s, subcommand, args)
	if err != nil {
		return err
	}

	if !ok {
		return s.ReplyUsage(ctx, "add|delete|restrict|rename ...")
	}

	return nil
}

func cmdListAddBroadcaster(ctx context.Context, s *session, cmd string, args string) error {
	return cmdListAdd(ctx, s, args, AccessLevelBroadcaster)
}

func cmdListAddModerator(ctx context.Context, s *session, cmd string, args string) error {
	return cmdListAdd(ctx, s, args, AccessLevelModerator)
}

func cmdListAddSubscriber(ctx context.Context, s *session, cmd string, args string) error {
	return cmdListAdd(ctx, s, args, AccessLevelSubscriber)
}

func cmdListAddEveryone(ctx context.Context, s *session, cmd string, args string) error {
	return cmdListAdd(ctx, s, args, AccessLevelEveryone)
}

func cmdListAdd(ctx context.Context, s *session, args string, level AccessLevel) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<name>")
	}

	name, _ := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return usage()
	}

	if reservedCommandNames[name] {
		return s.Replyf(ctx, "List name '%s' is reserved.", name)
	}

	var warning string
	if isBuiltinName(name) {
		warning = " Warning: '" + name + "' is a builtin command and will now only be accessible via " + s.Channel.Prefix + "builtin " + name
	} else if prefixAndName, ok := isModerationCommand(s.Channel.Prefix, name); ok {
		warning = " Warning: '" + prefixAndName + "' is a moderation command; your list may not work."
	}

	info, list, err := findCommandList(ctx, s, name)
	if err != nil {
		return err
	}

	if info != nil {
		if list == nil {
			return s.Replyf(ctx, "A command or list with name '%s' already exists.", name)
		}
		return s.Replyf(ctx, "List '%s' already exists. Use %s%s add|delete|get|... to access it.", name, s.Channel.Prefix, name)
	}

	if !s.UserLevel.CanAccess(level) {
		return s.Replyf(ctx, "Your level is %s; you cannot add a list with level %s.", s.UserLevel.PGEnum(), level.PGEnum())
	}

	list = &models.CommandList{
		ChannelID: s.Channel.ID,
	}

	if err := list.Insert(ctx, s.Tx, boil.Infer()); err != nil {
		return err
	}

	info = &models.CommandInfo{
		ChannelID:     s.Channel.ID,
		Name:          name,
		AccessLevel:   level.PGEnum(),
		Creator:       s.User,
		Editor:        s.User,
		CommandListID: null.Int64From(list.ID),
	}

	if err := info.Insert(ctx, s.Tx, boil.Infer()); err != nil {
		return err
	}

	al := flect.Pluralize(info.AccessLevel)
	return s.Replyf(ctx, "List '%s' added, restricted to %s and above.%s", name, al, warning)
}

func cmdListDelete(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<name>")
	}

	name, _ := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return usage()
	}

	info, list, err := findCommandList(ctx, s, name)
	if err != nil {
		return err
	}

	if info == nil {
		return s.Replyf(ctx, "List '%s' does not exist.", name)
	}

	if list == nil {
		return s.Replyf(ctx, "'%s' is not a list.", name)
	}

	level := newAccessLevel(info.AccessLevel)
	if !s.UserLevel.CanAccess(level) {
		return s.Replyf(ctx, "Your level is %s; you cannot delete a list with level %s.", s.UserLevel.PGEnum(), info.AccessLevel)
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
		return s.Replyf(ctx, "List '%s' and its repeat/schedule have been deleted.", name)
	}

	return s.Replyf(ctx, "List '%s' deleted.", name)
}

func cmdListRestrict(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<name> everyone|regulars|subs|vips|mods|broadcaster|admin")
	}

	name, level := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return usage()
	}

	info, err := s.Channel.CommandInfos(models.CommandInfoWhere.Name.EQ(name), qm.For("UPDATE")).One(ctx, s.Tx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return s.Replyf(ctx, "List '%s' does not exist.", name)
		}
		return err
	}

	if !info.CommandListID.Valid {
		return s.Replyf(ctx, "'%s' is not a list.", name)
	}

	return handleListRestrict(ctx, s, info, level, usage)
}

func cmdListRename(ctx context.Context, s *session, cmd string, args string) error {
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
		if errors.Is(err, sql.ErrNoRows) {
			return s.Replyf(ctx, "List '%s' does not exist.", oldName)
		}
		return err
	}

	if !info.CommandListID.Valid {
		return s.Replyf(ctx, "'%s' is not a list.", oldName)
	}

	level := newAccessLevel(info.AccessLevel)
	if !s.UserLevel.CanAccess(level) {
		return s.Replyf(ctx, "Your level is %s; you cannot rename a list with level %s.", s.UserLevel.PGEnum(), info.AccessLevel)
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

	return s.Replyf(ctx, "List '%s' has been renamed to '%s'.", oldName, newName)
}

func findCommandList(ctx context.Context, s *session, name string) (*models.CommandInfo, *models.CommandList, error) {
	info, err := s.Channel.CommandInfos(
		models.CommandInfoWhere.Name.EQ(name),
		qm.Load(models.CommandInfoRels.CommandList, qm.For("UPDATE")),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	return info, info.R.CommandList, nil
}

func handleList(ctx context.Context, s *session, info *models.CommandInfo, update bool) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "handleList")
	defer span.End()

	args := s.CommandParams
	cmd, args := splitSpace(args)
	cmd = strings.ToLower(cmd)

	switch cmd {
	case "add", "delete", "remove", "rm", "restrict":
		if !s.UserLevel.CanAccess(AccessLevelModerator) {
			return true, errNotAuthorized
		}

		if !update {
			return true, s.Reply(ctx, "Cross-channel commands may not modify lists.")
		}
	}

	if err := s.TryCooldown(ctx); err != nil {
		return false, err
	}

	defer s.UsageContext(info.Name)()

	random := false

	switch cmd {
	case "add":
		return true, handleListAdd(ctx, s, info, args)
	case "delete", "remove", "rm":
		return true, handleListDelete(ctx, s, info, cmd, args)
	case "restrict":
		return true, handleListRestrict(ctx, s, info, args, func() error {
			return s.ReplyUsage(ctx, "restrict everyone|regulars|subs|vips|mods|broadcaster|admin")
		})
	case "random", "":
		random = true
	case "get":
		cmd, args = splitSpace(args)
	}

	var num int

	if !random {
		var err error
		num, err = strconv.Atoi(cmd)
		if err != nil {
			return true, s.ReplyUsage(ctx, "get <index>")
		}

		num--

		if num < 0 {
			return true, s.Reply(ctx, "Index out of range.")
		}
	}

	list, err := info.CommandList().One(ctx, s.Tx)
	if err != nil {
		return true, err
	}

	if len(list.Items) == 0 {
		if random {
			return false, nil
		}
		return true, s.Reply(ctx, "The list is empty.")
	}

	if random {
		num = s.Deps.Rand.Intn(len(list.Items))
	} else if num >= len(list.Items) {
		return true, s.Reply(ctx, "Index out of range.")
	}

	item := list.Items[num]

	s.SetCommandParams(args)

	return true, runCommandAndCount(ctx, s, info, item, update)
}

func handleListRestrict(ctx context.Context, s *session, info *models.CommandInfo, level string, usage func() error) error {
	if level == "" {
		return s.Replyf(ctx, "List '%s' is restricted to %s and above.", info.Name, flect.Pluralize(info.AccessLevel))
	}

	level = strings.ToLower(level)

	newLevel := parseLevelPG(level)
	if newLevel == "" {
		return usage()
	}

	if !s.UserLevel.CanAccess(newAccessLevel(info.AccessLevel)) {
		return s.Replyf(ctx, "Your level is %s; you cannot restrict a list with level %s.", s.UserLevel.PGEnum(), info.AccessLevel)
	}

	if !s.UserLevel.CanAccess(newAccessLevel(newLevel)) {
		return s.Replyf(ctx, "Your level is %s; you cannot restrict a list to level %s.", s.UserLevel.PGEnum(), newLevel)
	}

	info.AccessLevel = newLevel
	info.Editor = s.User

	if err := info.Update(ctx, s.Tx, boil.Whitelist(models.CommandInfoColumns.UpdatedAt, models.CommandInfoColumns.AccessLevel, models.CommandInfoColumns.Editor)); err != nil {
		return err
	}

	return s.Replyf(ctx, "List '%s' restricted to %s and above.", info.Name, flect.Pluralize(info.AccessLevel))
}

func handleListAdd(ctx context.Context, s *session, info *models.CommandInfo, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "add <item>")
	}

	list, err := info.CommandList(qm.For("UPDATE")).One(ctx, s.Tx)
	if err != nil {
		return err
	}

	_, exists := stringSliceIndex(list.Items, args)
	if exists {
		return s.Reply(ctx, "The list already contains that item.")
	}

	var warning string
	if _, malformed := cbp.Parse(args); malformed {
		warning += " Warning: item contains stray (_ or _) separators and may not be processed correctly."
	}

	list.Items = append(list.Items, args)

	if err := list.Update(ctx, s.Tx, boil.Infer()); err != nil {
		return err
	}

	info.Editor = s.User

	if err := info.Update(ctx, s.Tx, boil.Whitelist(models.CommandInfoColumns.UpdatedAt, models.CommandInfoColumns.Editor)); err != nil {
		return err
	}

	return s.Replyf(ctx, `"%s" has been added to the list as item #%d.%s`, args, len(list.Items), warning)
}

func handleListDelete(ctx context.Context, s *session, info *models.CommandInfo, cmd, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, cmd+" <num>")
	}

	idxStr, _ := splitSpace(args)
	i, err := strconv.Atoi(idxStr)
	if err != nil {
		return usage()
	}

	i--

	if i < 0 {
		return usage()
	}

	list, err := info.CommandList().One(ctx, s.Tx)
	if err != nil {
		return err
	}

	if i >= len(list.Items) {
		return usage()
	}

	removed := list.Items[i]
	list.Items = append(list.Items[:i], list.Items[i+1:]...)

	if err := list.Update(ctx, s.Tx, boil.Infer()); err != nil {
		return err
	}

	info.Editor = s.User

	if err := info.Update(ctx, s.Tx, boil.Whitelist(models.CommandInfoColumns.UpdatedAt, models.CommandInfoColumns.Editor)); err != nil {
		return err
	}

	return s.Replyf(ctx, `"%s" has been removed.`, removed)
}
