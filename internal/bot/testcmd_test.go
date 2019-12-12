package bot_test

import (
	"context"
	"encoding/json"
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
			return s.Reply(ctx, s.UserLevel.String())
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_name",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return s.Reply(ctx, s.User)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_display_name",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return s.Reply(ctx, s.UserDisplay)
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

			links := s.Links(ctx)
			last := len(links) - 1

			for i, link := range links {
				builder.WriteString(link.String())

				if i != last {
					builder.WriteByte(' ')
				}
			}

			return s.Reply(ctx, builder.String())
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_bad_command",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return s.SendCommand(ctx, "fake")
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_delete",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return s.DeleteMessage(ctx)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_message_count",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return s.Replyf(ctx, "%v", s.N)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_twitch_token",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			tok, err := s.TwitchToken(ctx)
			if err != nil {
				return err
			}

			// pgx converts times back to the client timezone; convert to UTC for testing.
			tok.Expiry = tok.Expiry.UTC()

			j, err := json.Marshal(tok)
			if err != nil {
				return err
			}

			return s.Replyf(ctx, "%s", j)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_user_state",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			fast, err := s.Deps.Redis.GetUserState(ctx, s.Origin, "#"+s.IRCChannel)
			if err != nil {
				return err
			}
			return s.Replyf(ctx, "fast=%v", fast)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_channel_display_name",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return s.Reply(ctx, s.Channel.DisplayName)
		},
		bot.LevelEveryone,
	)
}
