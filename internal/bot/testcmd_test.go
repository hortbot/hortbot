package bot_test

import (
	"context"
	"fmt"

	"github.com/hortbot/hortbot/internal/bot"
)

func init() {
	bot.AddBuiltin("testing_panic",
		func(ctx context.Context, s *bot.Session, args string) error {
			panic(args)
		},
		bot.LevelEveryone,
	)

	bot.AddBuiltin("testing_access_level",
		func(ctx context.Context, s *bot.Session, args string) error {
			return s.Reply(s.UserLevel.String())
		},
		bot.LevelEveryone,
	)

	bot.AddBuiltin("testing_name",
		func(ctx context.Context, s *bot.Session, args string) error {
			return s.Reply(s.User)
		},
		bot.LevelEveryone,
	)

	bot.AddBuiltin("testing_display_name",
		func(ctx context.Context, s *bot.Session, args string) error {
			return s.Reply(s.UserDisplay)
		},
		bot.LevelEveryone,
	)

	bot.AddBuiltin("testing_error",
		func(ctx context.Context, s *bot.Session, args string) error {
			return fmt.Errorf("%s", args)
		},
		bot.LevelEveryone,
	)
}
