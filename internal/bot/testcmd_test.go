package bot_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

func init() {
	bot.TestingAction(func(_ context.Context, action string) (string, error, bool) {
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
		func(ctx context.Context, s *bot.Session, _ string, _ string) error {
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

	bot.TestingBuiltin("testing_delete",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return s.DeleteMessage(ctx)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_message_count",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return s.Replyf(ctx, "%v", s.Channel.MessageCount)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_twitch_token",
		func(ctx context.Context, s *bot.Session, _ string, _ string) error {
			tok, err := s.ChannelTwitchToken(ctx)
			if err != nil {
				return fmt.Errorf("getting twitch token: %w", err)
			}

			// pgx converts times back to the client timezone; convert to UTC for testing.
			tok.Expiry = tok.Expiry.UTC()

			j, err := json.Marshal(tok)
			if err != nil {
				return fmt.Errorf("marshalling token: %w", err)
			}

			return s.Replyf(ctx, "%s", j)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_channel_display_name",
		func(ctx context.Context, s *bot.Session, cmd string, args string) error {
			return s.Reply(ctx, s.Channel.DisplayName)
		},
		bot.LevelEveryone,
	)

	bot.TestingBuiltin("testing_highlights",
		func(ctx context.Context, s *bot.Session, _ string, _ string) error {
			highlights, err := s.Channel.Highlights(qm.OrderBy(models.HighlightColumns.CreatedAt)).All(ctx, s.Tx)
			if err != nil {
				return fmt.Errorf("getting highlights: %w", err)
			}

			if len(highlights) == 0 {
				return s.Reply(ctx, "No highlights.")
			}

			var builder strings.Builder
			for i, h := range highlights {
				if i != 0 {
					builder.WriteByte(' ')
				}

				startedAt := h.StartedAt.Ptr()
				if startedAt != nil {
					x := startedAt.UTC()
					startedAt = &x
				}

				fmt.Fprintf(&builder, "[%v, %v, %q, %q]", h.HighlightedAt.UTC(), startedAt, h.Status, h.Game)
			}

			return s.Reply(ctx, builder.String())
		},
		bot.LevelEveryone,
	)
}
