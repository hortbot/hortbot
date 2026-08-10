package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

func tryAutoreplies(ctx context.Context, s *session) (bool, error) {
	autoreplies, err := s.Queries.ListAutoreplyMatchers(ctx, s.Channel.ID)
	if err != nil {
		return true, fmt.Errorf("querying for autoreplies: %w", err)
	}

	for _, autoreply := range autoreplies {
		re, err := s.Deps.ReCache.Compile(autoreply.Trigger)
		if err != nil {
			ctxlog.Warn(ctx, "failed to compile regex", zap.Error(err))
			continue
		}

		if !re.MatchString(s.Message) {
			continue
		}

		msg := autoreply.Response

		if strings.Contains(msg, "(_REGULARS_ONLY_)") {
			if !s.UserLevel.CanAccess(AccessLevelSubscriber) {
				continue
			}
			msg = strings.ReplaceAll(msg, "(_REGULARS_ONLY_)", "")
		}

		allowed, err := s.AutoreplyAllowed(ctx, autoreply.ID, 30*time.Second)
		if err != nil {
			return true, err
		}

		if !allowed {
			// Allow further autoreplies to match.
			continue
		}

		autoreply.Count++
		if err := s.Queries.UpdateAutoreplyCount(ctx, dbsql.UpdateAutoreplyCountParams{
			Count: autoreply.Count,
			ID:    autoreply.ID,
		}); err != nil {
			return true, fmt.Errorf("updating autoreply count: %w", err)
		}

		oldType := s.Type
		s.Type = sessionAutoreply
		defer func() {
			s.Type = oldType
		}()

		reply, err := processCommand(ctx, s, msg)
		if err != nil {
			return true, err
		}

		if err := s.Reply(ctx, reply); err != nil {
			return true, err
		}

		metricAutoreplies.Inc()

		return true, nil
	}

	return false, nil
}
