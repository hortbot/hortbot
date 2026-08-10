package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/jackc/pgx/v5/pgtype"
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

	insertedList, err := s.Queries.InsertCommandList(ctx, s.Channel.ID)
	if err != nil {
		return fmt.Errorf("inserting list: %w", err)
	}

	insertedInfo, err := s.Queries.InsertCommandInfo(ctx, dbsql.InsertCommandInfoParams{
		ChannelID:       s.Channel.ID,
		Name:            name,
		AccessLevel:     level.PGEnum(),
		Creator:         s.User,
		Editor:          s.User,
		CustomCommandID: pgtype.Int8{},
		CommandListID:   dbsql.Int8From(insertedList.ID),
	})
	if err != nil {
		return fmt.Errorf("inserting command info: %w", err)
	}

	al := pluralAccessLevel(insertedInfo.AccessLevel)
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

	repeated, scheduled, err := s.Queries.DeleteCommandInfoCascade(ctx, info)
	if err != nil {
		return fmt.Errorf("deleting command info: %w", err)
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

	info, _, found, err := s.Queries.LookupCommand(ctx, s.Channel.ID, name, true)
	if err != nil {
		return fmt.Errorf("getting command info: %w", err)
	}
	if !found {
		return s.Replyf(ctx, "List '%s' does not exist.", name)
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

	info, _, found, err := s.Queries.LookupCommand(ctx, s.Channel.ID, oldName, true)
	if err != nil {
		return fmt.Errorf("getting command info: %w", err)
	}
	if !found {
		return s.Replyf(ctx, "List '%s' does not exist.", oldName)
	}

	if !info.CommandListID.Valid {
		return s.Replyf(ctx, "'%s' is not a list.", oldName)
	}

	level := newAccessLevel(info.AccessLevel)
	if !s.UserLevel.CanAccess(level) {
		return s.Replyf(ctx, "Your level is %s; you cannot rename a list with level %s.", s.UserLevel.PGEnum(), info.AccessLevel)
	}

	exists, err := s.Queries.CommandInfoExists(ctx, dbsql.CommandInfoExistsParams{
		ChannelID: s.Channel.ID,
		Name:      newName,
	})
	if err != nil {
		return fmt.Errorf("checking command info exists: %w", err)
	}

	if exists {
		return s.Replyf(ctx, "A command or list with name '%s' already exists.", newName)
	}

	info.Name = newName
	info.Editor = s.User

	if err := s.Queries.RenameCommandInfo(ctx, dbsql.RenameCommandInfoParams{
		Name:   info.Name,
		Editor: info.Editor,
		ID:     info.ID,
	}); err != nil {
		return fmt.Errorf("updating command info: %w", err)
	}

	return s.Replyf(ctx, "List '%s' has been renamed to '%s'.", oldName, newName)
}

func findCommandList(ctx context.Context, s *session, name string) (*dbsql.CommandInfo, *dbsql.CommandList, error) {
	info, _, found, err := s.Queries.LookupCommand(ctx, s.Channel.ID, name, true)
	if err != nil {
		return nil, nil, fmt.Errorf("getting command info: %w", err)
	}
	if !found {
		return nil, nil, nil
	}
	if !info.CommandListID.Valid {
		return info, nil, nil
	}
	list, err := s.Queries.GetCommandListForUpdate(ctx, info.CommandListID.Int64)
	if err != nil {
		return nil, nil, fmt.Errorf("getting command list: %w", err)
	}
	return info, &dbsql.CommandList{
		ID:        list.ID,
		CreatedAt: list.CreatedAt,
		UpdatedAt: list.UpdatedAt,
		ChannelID: list.ChannelID,
		Items:     list.Items,
	}, nil
}

func handleList(ctx context.Context, s *session, info *dbsql.CommandInfo, update bool) (bool, error) {
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

	list, err := s.Queries.GetCommandList(ctx, info.CommandListID.Int64)
	if err != nil {
		return true, fmt.Errorf("getting command list: %w", err)
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

func handleListRestrict(ctx context.Context, s *session, info *dbsql.CommandInfo, level string, usage func() error) error {
	if level == "" {
		return s.Replyf(ctx, "List '%s' is restricted to %s and above.", info.Name, pluralAccessLevel(info.AccessLevel))
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

	if err := s.Queries.UpdateCommandInfoAccess(ctx, dbsql.UpdateCommandInfoAccessParams{
		AccessLevel: info.AccessLevel,
		Editor:      info.Editor,
		ID:          info.ID,
	}); err != nil {
		return fmt.Errorf("updating command info: %w", err)
	}

	return s.Replyf(ctx, "List '%s' restricted to %s and above.", info.Name, pluralAccessLevel(info.AccessLevel))
}

func handleListAdd(ctx context.Context, s *session, info *dbsql.CommandInfo, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "add <item>")
	}

	list, err := s.Queries.GetCommandListForUpdate(ctx, info.CommandListID.Int64)
	if err != nil {
		return fmt.Errorf("getting command list: %w", err)
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

	if err := s.Queries.UpdateCommandListItems(ctx, dbsql.UpdateCommandListItemsParams{
		Items: list.Items,
		ID:    list.ID,
	}); err != nil {
		return fmt.Errorf("updating command list: %w", err)
	}

	info.Editor = s.User

	if err := s.Queries.UpdateCommandInfoEditor(ctx, dbsql.UpdateCommandInfoEditorParams{
		Editor: info.Editor,
		ID:     info.ID,
	}); err != nil {
		return fmt.Errorf("updating command info: %w", err)
	}

	return s.Replyf(ctx, `"%s" has been added to the list as item #%d.%s`, args, len(list.Items), warning)
}

func handleListDelete(ctx context.Context, s *session, info *dbsql.CommandInfo, cmd, args string) error {
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

	list, err := s.Queries.GetCommandListForUpdate(ctx, info.CommandListID.Int64)
	if err != nil {
		return fmt.Errorf("getting command list: %w", err)
	}

	if i >= len(list.Items) {
		return usage()
	}

	removed := list.Items[i]
	list.Items = append(list.Items[:i], list.Items[i+1:]...)

	if err := s.Queries.UpdateCommandListItems(ctx, dbsql.UpdateCommandListItemsParams{
		Items: list.Items,
		ID:    list.ID,
	}); err != nil {
		return fmt.Errorf("updating command list: %w", err)
	}

	info.Editor = s.User

	if err := s.Queries.UpdateCommandInfoEditor(ctx, dbsql.UpdateCommandInfoEditorParams{
		Editor: info.Editor,
		ID:     info.ID,
	}); err != nil {
		return fmt.Errorf("updating command info: %w", err)
	}

	return s.Replyf(ctx, `"%s" has been removed.`, removed)
}
