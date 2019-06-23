package bot_test

import (
	"context"
	"fmt"
	"strings"

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
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			panic(args)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_access_level",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return s.Reply(s.UserLevel.String())
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_name",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return s.Reply(s.User)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_display_name",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return s.Reply(s.UserDisplay)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_error",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return fmt.Errorf("%s", args)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_disabled",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return bot.ErrBuiltinDisabled
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_links",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			var builder strings.Builder
			builder.WriteString("Links: ")

			links := s.Links()
			last := len(links) - 1

			for i, link := range links {
				builder.WriteString(link.String())

				if i != last {
					builder.WriteByte(' ')
				}
			}

			return s.Reply(builder.String())
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_bad_command",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return s.SendCommand("fake")
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_delete",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return s.DeleteMessage()
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_message_count",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return s.Replyf("%v", s.N)
		},
		bot.LevelEveryone,
	)
}
