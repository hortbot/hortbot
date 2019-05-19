package bot_test

import (
	"context"

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
}
