package bot_test

import (
	"context"
	"fmt"

	"github.com/hortbot/hortbot/internal/bot"
)

func init() {
	bot.TestingAction(func(ctx context.Context, action string) (string, error, bool) {
		if action == "TESTING_ERROR" {
			return "(error)", fmt.Errorf("%s", action), true
		}

		return "", nil, false
	})

	bot.TestingBuiltin("testing_panic",
		func(ctx context.Context, s *bot.Session, args string) error {
			panic(args)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_access_level",
		func(ctx context.Context, s *bot.Session, args string) error {
			return s.Reply(s.UserLevel.String())
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_name",
		func(ctx context.Context, s *bot.Session, args string) error {
			return s.Reply(s.User)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_display_name",
		func(ctx context.Context, s *bot.Session, args string) error {
			return s.Reply(s.UserDisplay)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_error",
		func(ctx context.Context, s *bot.Session, args string) error {
			return fmt.Errorf("%s", args)
		},
		bot.LevelEveryone,
	)
}
