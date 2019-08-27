package bot

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

// TODO: Merge the code between custom commands and lists; they are identical other than some wordings.

var listCommands = newHandlerMap(map[string]handlerFunc{
	"add":             {fn: cmdListAddSubscriber, minLevel: levelModerator},
	"addb":            {fn: cmdListAddBroadcaster, minLevel: levelModerator},
	"addbroadcaster":  {fn: cmdListAddBroadcaster, minLevel: levelModerator},
	"addbroadcasters": {fn: cmdListAddBroadcaster, minLevel: levelModerator},
	"addo":            {fn: cmdListAddBroadcaster, minLevel: levelModerator},
	"addowner":        {fn: cmdListAddBroadcaster, minLevel: levelModerator},
	"addowners":       {fn: cmdListAddBroadcaster, minLevel: levelModerator},
	"addstreamer":     {fn: cmdListAddBroadcaster, minLevel: levelModerator},
	"addstreamers":    {fn: cmdListAddBroadcaster, minLevel: levelModerator},
	"addm":            {fn: cmdListAddModerator, minLevel: levelModerator},
	"addmod":          {fn: cmdListAddModerator, minLevel: levelModerator},
	"addmods":         {fn: cmdListAddModerator, minLevel: levelModerator},
	"adds":            {fn: cmdListAddSubscriber, minLevel: levelModerator},
	"addsub":          {fn: cmdListAddSubscriber, minLevel: levelModerator},
	"addsubs":         {fn: cmdListAddSubscriber, minLevel: levelModerator},
	"adde":            {fn: cmdListAddEveryone, minLevel: levelModerator},
	"adda":            {fn: cmdListAddEveryone, minLevel: levelModerator},
	"addeveryone":     {fn: cmdListAddEveryone, minLevel: levelModerator},
	"addall":          {fn: cmdListAddEveryone, minLevel: levelModerator},
	"delete":          {fn: cmdListDelete, minLevel: levelModerator},
	"remove":          {fn: cmdListDelete, minLevel: levelModerator},
	"restrict":        {fn: cmdListRestrict, minLevel: levelModerator},
	"rename":          {fn: cmdListRename, minLevel: levelModerator},
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
	return cmdListAdd(ctx, s, args, levelBroadcaster)
}

func cmdListAddModerator(ctx context.Context, s *session, cmd string, args string) error {
	return cmdListAdd(ctx, s, args, levelModerator)
}

func cmdListAddSubscriber(ctx context.Context, s *session, cmd string, args string) error {
	return cmdListAdd(ctx, s, args, levelSubscriber)
}

func cmdListAddEveryone(ctx context.Context, s *session, cmd string, args string) error {
	return cmdListAdd(ctx, s, args, levelEveryone)
}

func cmdListAdd(ctx context.Context, s *session, args string, level accessLevel) error {
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

	// TODO: remove this warning
	var warning string
	if isBuiltinName(name) {
		warning = " Warning: '" + name + "' is a builtin command and will now only be accessible via " + s.Channel.Prefix + "builtin " + name
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
		s.Deps.UpdateRepeat(repeated.ID, false, 0, 0)
	}

	if scheduled != nil {
		deletedRepeat = true
		s.Deps.UpdateSchedule(scheduled.ID, false, nil)
	}

	if deletedRepeat {
		return s.Replyf(ctx, "List '%s' and its repeat/schedule have been deleted.", name)
	}

	return s.Replyf(ctx, "List '%s' deleted.", name)
}

func cmdListRestrict(ctx context.Context, s *session, cmd string, args string) error {
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
		if err == sql.ErrNoRows {
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

	switch err {
	case nil:
		return info, info.R.CommandList, nil
	case sql.ErrNoRows:
		return nil, nil, nil
	default:
		return nil, nil, err
	}
}

func handleList(ctx context.Context, s *session, info *models.CommandInfo, update bool) (bool, error) {
	args := s.OrigCommandParams
	cmd, args := splitSpace(args)
	cmd = strings.ToLower(cmd)

	if !s.UserLevel.CanAccess(levelModerator) {
		switch cmd {
		case "add", "delete", "remove", "restrict":
			return true, errNotAuthorized
		}
	}

	if err := s.TryCooldown(); err != nil {
		return false, err
	}

	defer s.UsageContext(info.Name)()

	random := false

	switch cmd {
	case "add":
		return true, handleListAdd(ctx, s, info, args)
	case "delete", "remove":
		return true, handleListDelete(ctx, s, info, cmd, args)
	case "restrict":
		return true, handleListRestrict(ctx, s, info, args, func() error {
			return s.ReplyUsage(ctx, "restrict everyone|regulars|subs|mods|broadcaster|admin")
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

	s.CommandParams = args
	s.OrigCommandParams = args

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

	_, err = cbp.Parse(args)
	if err != nil {
		return s.Replyf(ctx, "Error parsing list item.")
	}

	list.Items = append(list.Items, args)

	if err := list.Update(ctx, s.Tx, boil.Infer()); err != nil {
		return err
	}

	info.Editor = s.User

	if err := info.Update(ctx, s.Tx, boil.Whitelist(models.CommandInfoColumns.UpdatedAt, models.CommandInfoColumns.Editor)); err != nil {
		return err
	}

	return s.Replyf(ctx, `"%s" has been added to the list as item #%d.`, args, len(list.Items))
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
