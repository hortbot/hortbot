package bot

import "context"

func cmdChannelID(ctx context.Context, s *session, _ string, _ string) error {
	return s.Replyf(ctx, "%s's ID: %d, your ID: %d", s.Channel.DisplayName, s.RoomID, s.UserID)
}
