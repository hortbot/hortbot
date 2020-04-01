package bot

import (
	"context"
	"errors"
	"strings"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
)

func cmdChannelID(ctx context.Context, s *session, _ string, _ string) error {
	return s.Replyf(ctx, "%s's ID: %d, your ID: %d", s.Channel.DisplayName, s.RoomID, s.UserID)
}

func cmdHLTB(ctx context.Context, s *session, _ string, args string) error {
	query := args
	if query == "" {
		channel, err := s.TwitchChannel(ctx)
		if err != nil {
			return s.Reply(ctx, twitchServerErrorReply)
		}

		if channel.Game == "" {
			return s.Reply(ctx, "No game set.")
		}

		query = channel.Game
	}

	game, err := s.Deps.HLTB.SearchGame(ctx, query)
	if err != nil {
		var apiErr *apiclient.Error
		if errors.As(err, &apiErr) {
			if apiErr.IsNotFound() {
				return s.Replyf(ctx, "%s not found on HowLongToBeat.", query)
			}
			return s.Reply(ctx, "A HowLongToBeat API error occurred.")
		}
	}

	var b strings.Builder
	b.WriteString("HowLongToBeat for ")
	b.WriteString(game.Title)
	b.WriteString(" - ")
	b.WriteString(game.MainStory)
	b.WriteString(" main story")

	if game.MainPlusExtra != "" {
		b.WriteString(", ")
		b.WriteString(game.MainPlusExtra)
		b.WriteString(" main story + extra")
	}

	if game.Completionist != "" {
		b.WriteString(", ")
		b.WriteString(game.Completionist)
		b.WriteString(" completionist")
	}

	return s.Reply(ctx, b.String())
}
