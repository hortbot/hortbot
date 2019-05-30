package bot

import (
	"context"
	"time"
)

func cmdRoundtrip(ctx context.Context, s *Session, cmd string, args string) error {
	now := time.Now()
	return s.Replyf("total=%v, handle=%v", now.Sub(s.TMISent), now.Sub(s.Start))
}
