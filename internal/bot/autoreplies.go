package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

func tryAutoreplies(ctx context.Context, s *session) (bool, error) {
	var autoreplies models.AutoreplySlice
	err := queries.Raw(`
		SELECT autoreplies.id, autoreplies.trigger, autoreplies.response, autoreplies.count
		FROM autoreplies
		WHERE autoreplies.channel_id = $1
		ORDER BY autoreplies.num ASC
		`, s.Channel.ID).Bind(ctx, s.Tx, &autoreplies)
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
		if err := autoreply.Update(ctx, s.Tx, boil.Whitelist(models.AutoreplyColumns.Count)); err != nil {
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
