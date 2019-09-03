package bot

import (
	"context"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/opentracing/opentracing-go"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
)

func tryAutoreplies(ctx context.Context, s *session) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "tryAutoreplies")
	defer span.Finish()

	var autoreplies models.AutoreplySlice
	err := queries.Raw(`
		SELECT autoreplies.id, autoreplies.trigger, autoreplies.response, autoreplies.count
		FROM autoreplies
		WHERE autoreplies.channel_id = $1
		ORDER BY autoreplies.num ASC
		`, s.Channel.ID).Bind(ctx, s.Tx, &autoreplies)
	if err != nil {
		return true, err
	}

	// TODO: Keep local cache of autoreplies per channel, instead of just the Regexps themselves.
	for _, autoreply := range autoreplies {
		re, err := s.Deps.ReCache.Compile(autoreply.Trigger)
		if err != nil {
			continue
		}

		if !re.MatchString(s.Message) {
			continue
		}

		msg := autoreply.Response

		if strings.Contains(msg, "(_REGULARS_ONLY_)") {
			if !s.UserLevel.CanAccess(levelSubscriber) {
				continue
			}
			msg = strings.ReplaceAll(msg, "(_REGULARS_ONLY_)", "")
		}

		allowed, err := s.AutoreplyAllowed(ctx, autoreply.ID, 30*time.Second)
		if err != nil {
			return true, err
		}

		if !allowed {
			return true, nil
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

		autoreply.Count++

		return true, autoreply.Update(ctx, s.Tx, boil.Whitelist(models.AutoreplyColumns.Count))
	}

	return false, nil
}
