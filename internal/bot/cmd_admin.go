package bot

import (
	"context"
)

func cmdRoundtrip(ctx context.Context, s *Session, cmd string, args string) error {
	now := s.Clock.Now()
	return s.Replyf("total=%v, handle=%v", now.Sub(s.TMISent), now.Sub(s.Start))
}
