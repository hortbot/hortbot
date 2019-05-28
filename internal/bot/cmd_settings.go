package bot

import (
	"context"
	"sort"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
)

func cmdBullet(ctx context.Context, s *Session, cmd string, args string) error {
	args = strings.TrimSpace(args)

	if args == "" {
		var bullet string
		if s.Channel.Bullet.Valid {
			bullet = s.Channel.Bullet.String
		} else {
			bullet = s.Bot.bullet + " (default)"
		}

		return s.Replyf("bullet is %s", bullet)
	}

	switch args[0] {
	case '/', '.':
		return s.Reply("bullet cannot be a command")
	}

	reset := strings.EqualFold(args, "reset")

	if reset {
		s.Channel.Bullet = null.String{}
	} else {
		s.Channel.Bullet = null.StringFrom(args)
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Bullet)); err != nil {
		return err
	}

	if reset {
		return s.Reply("bullet reset to default")
	}

	return s.Replyf("bullet changed to %s", args)
}

func cmdPrefix(ctx context.Context, s *Session, cmd string, args string) error {
	args = strings.TrimSpace(args)

	if args == "" {
		return s.Replyf("prefix is %s", s.Channel.Prefix)
	}

	switch args[0] {
	case '/', '.':
		return s.Replyf("prefix cannot begin with %c", args[0])
	}

	reset := strings.EqualFold(args, "reset")

	if reset {
		s.Channel.Prefix = s.Bot.prefix
	} else {
		s.Channel.Prefix = args
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Prefix)); err != nil {
		return err
	}

	if reset {
		return s.Replyf("prefix reset to %s", s.Channel.Prefix)
	}

	return s.Replyf("prefix changed to %s", args)
}

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
		return s.ReplyUsage(cmd + " <list|add|remove> ...")
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
			return s.Replyf("there are no %s", cmds)
		}

		sort.Strings(existing)

		return s.Replyf("%s: %s", cmds, strings.Join(existing, ", "))

	case "add":
		if _, found := stringSliceIndex(existing, user); found {
			return s.Replyf("%s is already in %s", user, cmds)
		}

		existing = append(existing, user)

		if err := setter(existing); err != nil {
			return err
		}

		return s.Replyf("%s added to %s", user, cmds)

	case "remove", "delete":
		i, found := stringSliceIndex(existing, user)
		if !found {
			return s.Replyf("%s is not in %s", user, cmds)
		}

		existing[i] = existing[len(existing)-1]
		existing = existing[:len(existing)-1]

		if err := setter(existing); err != nil {
			return err
		}

		return s.Replyf("%s removed from %s", user, cmds)
	}

	return usage()
}
