package bot

import (
	"context"
)

func cmdRoundtrip(ctx context.Context, s *session, cmd string, args string) error {
	now := s.Deps.Clock.Now()
	return s.Replyf("total=%v, handle=%v", now.Sub(s.TMISent), now.Sub(s.Start))
}
