package bot

import (
	"context"
	"database/sql"

	"github.com/hortbot/hortbot/internal/db/models"
)

func tryFilter(ctx context.Context, s *Session) (filtered bool, err error) {
	if s.Channel.FilterLinks {
		filtered, err = filterLinks(ctx, s)
		if filtered || err != nil {
			return filtered, err
		}
	}

	return false, nil
}

func filterLinks(ctx context.Context, s *Session) (filtered bool, err error) {
	if len(s.Links()) == 0 {
		return false, nil
	}

	if s.UserLevel.CanAccess(LevelSubscriber) {
		return false, nil
	}

	permit, err := models.LinkPermits(
		models.LinkPermitWhere.ChannelID.EQ(s.Channel.ID),
		models.LinkPermitWhere.Name.EQ(s.User),
	).One(ctx, s.Tx)

	switch err {
	case nil:
		if err := permit.Delete(ctx, s.Tx); err != nil {
			return false, err
		}

		if s.Clock.Now().Before(permit.ExpiresAt) {
			return false, s.Replyf("Link permitted (%s)", s.UserDisplay)
		}
	case sql.ErrNoRows:
		// Fall through
	default:
		return false, err
	}

	if err := s.DeleteMessage(); err != nil {
		return true, err
	}

	return true, s.Replyf("%s, please ask a moderator before posting links.", s.UserDisplay)
}
