package bot

import (
	"context"
	"sort"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/boil"
)

func cmdOwnerModRegularIgnore(ctx context.Context, s *Session, cmd string, args string) error {
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
		return s.ReplyUsage("<list|add|remove> ...")
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
			return s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.CustomOwners))

		case "mod":
			s.Channel.CustomMods = v
			return s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.CustomMods))

		case "regular":
			s.Channel.CustomRegulars = v
			return s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.CustomRegulars))

		case "ignore":
			s.Channel.Ignored = v
			return s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Ignored))

		default:
			panic("unreachable")
		}
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
			return s.Replyf("There are no %s.", cmds)
		}

		sort.Strings(existing)

		return s.Replyf("%s: %s", cmds, strings.Join(existing, ", "))

	case "add":
		if _, found := stringSliceIndex(existing, user); found {
			return s.Replyf("%s is already in %s.", user, cmds)
		}

		existing = append(existing, user)

		if err := setter(existing); err != nil {
			return err
		}

		return s.Replyf("%s added to %s.", user, cmds)

	case "remove", "delete":
		i, found := stringSliceIndex(existing, user)
		if !found {
			return s.Replyf("%s is not in %s.", user, cmds)
		}

		existing[i] = existing[len(existing)-1]
		existing = existing[:len(existing)-1]

		if err := setter(existing); err != nil {
			return err
		}

		return s.Replyf("%s removed from %s.", user, cmds)
	}

	return usage()
}
