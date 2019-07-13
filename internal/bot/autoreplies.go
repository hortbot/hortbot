package bot

import (
	"context"
	"strings"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

// var autoreplyCommandRegex = regexp.MustCompile(`\(_COMMAND_(.*)_\)`)

func tryAutoreplies(ctx context.Context, s *session) (bool, error) {
	autoreplies, err := s.Channel.Autoreplies(
		qm.Select(models.AutoreplyColumns.ID, models.AutoreplyColumns.Trigger, models.AutoreplyColumns.Response),
		qm.OrderBy(models.AutoreplyColumns.Num),
	).All(ctx, s.Tx)
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

		allowed, err := s.AutoreplyAllowed(autoreply.ID, 30)
		if err != nil {
			return true, err
		}

		if !allowed {
			return true, nil
		}

		autoreply.Count++

		if err := autoreply.Update(ctx, s.Tx, boil.Whitelist(models.AutoreplyColumns.Count)); err != nil {
			return true, err
		}

		// TODO: Don't do this as string replacements; do it in the normal handling way.
		switch {
		case strings.Contains(msg, "(_PURGE_)"):
			// TODO: Just delete message?
			if err := s.SendCommand("timeout", s.User, "1"); err != nil {
				return true, err
			}
			msg = strings.ReplaceAll(msg, "(_PURGE_)", "")

		case strings.Contains(msg, "(_TIMEOUT_)"):
			if err := s.SendCommand("timeout", s.User); err != nil {
				return true, err
			}
			msg = strings.ReplaceAll(msg, "(_TIMEOUT_)", "")

		case strings.Contains(msg, "(_BAN_)"):
			if err := s.SendCommand("ban", s.User); err != nil {
				return true, err
			}
			msg = strings.ReplaceAll(msg, "(_BAN_)", "")

		default:
			// TODO: (_COMMAND_<command>_)
		}

		// TODO: Handle autoreply instead of returning it verbatim.

		msg = strings.TrimSpace(msg)
		if msg == "" {
			return true, nil
		}
		return true, s.Reply(msg)
	}

	return false, nil
}
