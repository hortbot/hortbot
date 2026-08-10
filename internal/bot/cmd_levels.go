package bot

import (
	"context"
	"slices"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/db/dbsql"
)

func cmdOwnerModRegularIgnore(ctx context.Context, s *session, cmd string, args string) error {
	args = strings.TrimSpace(args)

	var cmds string

	switch cmd {
	case "owner", "mod", "regular":
		cmds = flect.Pluralize(cmd)
	case "ignore":
		cmds = "ignored users"
	default:
		panic("unreachable: " + cmd)
	}

	usage := func() error {
		return s.ReplyUsage(ctx, "list|add|remove ...")
	}

	getter := func() []string {
		switch cmd {
		case "owner":
			return s.Channel.CustomOwners
		case "mod":
			return s.Channel.CustomMods
		case "regular":
			return s.Channel.CustomRegulars
		case "ignore":
			return s.Channel.Ignored
		default:
			panic("unreachable")
		}
	}

	setter := func(v []string) error {
		switch cmd {
		case "owner":
			s.Channel.CustomOwners = v
		case "mod":
			s.Channel.CustomMods = v
		case "regular":
			s.Channel.CustomRegulars = v
		case "ignore":
			s.Channel.Ignored = v
		default:
			panic("unreachable")
		}

		return s.Queries.UpdateChannelUserLists(ctx, dbsql.UpdateChannelUserListsParams{
			CustomOwners:   s.Channel.CustomOwners,
			CustomMods:     s.Channel.CustomMods,
			CustomRegulars: s.Channel.CustomRegulars,
			Ignored:        s.Channel.Ignored,
			ID:             s.Channel.ID,
		})
	}

	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	user, _ := splitSpace(args)
	user = strings.TrimPrefix(user, "@")
	user = strings.ToLower(user)

	existing := getter()

	switch subcommand {
	case "list":
		if len(existing) == 0 {
			return s.Replyf(ctx, "There are no %s.", cmds)
		}

		slices.Sort(existing)

		return s.Replyf(ctx, "%s: %s", cmds, strings.Join(existing, ", "))

	case "add":
		if _, found := stringSliceIndex(existing, user); found {
			return s.Replyf(ctx, "%s is already in %s.", user, cmds)
		}

		existing = append(existing, user)

		if err := setter(existing); err != nil {
			return err
		}

		return s.Replyf(ctx, "%s added to %s.", user, cmds)

	case "remove", "delete":
		i, found := stringSliceIndex(existing, user)
		if !found {
			return s.Replyf(ctx, "%s is not in %s.", user, cmds)
		}

		existing[i] = existing[len(existing)-1]
		existing = existing[:len(existing)-1]

		if err := setter(existing); err != nil {
			return err
		}

		return s.Replyf(ctx, "%s removed from %s.", user, cmds)
	}

	return usage()
}
