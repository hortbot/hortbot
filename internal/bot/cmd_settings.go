package bot

import (
	"context"
	"strings"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
)

func cmdBullet(ctx context.Context, s *Session, args string) error {
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

	reset := strings.EqualFold(args, "reset")

	if reset {
		s.Channel.Bullet = null.String{}
	} else {
		s.Channel.Bullet = null.StringFrom(args)
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.Bullet)); err != nil {
		return err
	}

	if reset {
		return s.Reply("bullet reset to default")
	}

	return s.Replyf("bullet changed to %s", args)
}

func cmdPrefix(ctx context.Context, s *Session, args string) error {
	args = strings.TrimSpace(args)

	if args == "" {
		return s.Replyf("prefix is %s", s.Channel.Prefix)
	}

	reset := strings.EqualFold(args, "reset")

	if reset {
		s.Channel.Prefix = s.Bot.prefix
	} else {
		s.Channel.Prefix = args
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.Prefix)); err != nil {
		return err
	}

	if reset {
		return s.Replyf("prefix reset to %s", s.Channel.Prefix)
	}

	return s.Replyf("prefix changed to %s", args)
}
