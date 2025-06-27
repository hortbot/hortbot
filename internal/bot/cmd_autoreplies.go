package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"
	"strconv"
	"strings"
	"testing"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/errorsx"
)

var autoreplyCommands = newHandlerMap(map[string]handlerFunc{
	"add":          {fn: cmdAutoreplyAdd, minLevel: AccessLevelModerator},
	"delete":       {fn: cmdAutoreplyDelete, minLevel: AccessLevelModerator},
	"remove":       {fn: cmdAutoreplyDelete, minLevel: AccessLevelModerator},
	"editresponse": {fn: cmdAutoreplyEditResponse, minLevel: AccessLevelModerator},
	"editpattern":  {fn: cmdAutoreplyEditPattern, minLevel: AccessLevelModerator},
	"edittrigger":  {fn: cmdAutoreplyEditPattern, minLevel: AccessLevelModerator},
	"list":         {fn: cmdAutoreplyList, minLevel: AccessLevelSubscriber},
	"compact":      {fn: cmdAutoreplyCompact, minLevel: AccessLevelModerator},
})

func cmdAutoreply(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "add|remove|editresponse|editpattern|list")
	}

	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	if subcommand == "" {
		return usage()
	}

	ok, err := autoreplyCommands.Run(ctx, s, subcommand, args)
	if !ok {
		return usage()
	}

	return err
}

func cmdAutoreplyAdd(ctx context.Context, s *session, cmd string, args string) error {
	pattern, response := splitSpace(args)

	if pattern == "" || response == "" {
		return s.ReplyUsage(ctx, "<pattern> <response>")
	}

	trigger, err := s.patternToTrigger(pattern)
	if err != nil {
		return s.replyBadPattern(ctx, err)
	}

	var warning string
	if _, malformed := cbp.Parse(response); malformed {
		warning += " Warning: response contains stray (_ or _) separators and may not be processed correctly."
	}

	var row struct {
		MaxNum null.Int
	}

	err = s.Channel.Autoreplies(
		qm.Select("max("+models.AutoreplyColumns.Num+") as max_num"),
	).Bind(ctx, s.Tx, &row)
	if err != nil {
		return fmt.Errorf("getting max autoreply num: %w", err)
	}

	nextNum := row.MaxNum.Int + 1

	autoreply := &models.Autoreply{
		ChannelID:   s.Channel.ID,
		Num:         nextNum,
		Trigger:     trigger,
		OrigPattern: null.StringFrom(pattern),
		Response:    response,
		Creator:     s.User,
		Editor:      s.User,
	}

	if err := autoreply.Insert(ctx, s.Tx, boil.Infer()); err != nil {
		return fmt.Errorf("inserting autoreply: %w", err)
	}

	return s.Replyf(ctx, "Autoreply #%d added.%s", autoreply.Num, warning)
}

func cmdAutoreplyDelete(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<index>")
	}

	if args == "" {
		return usage()
	}

	num, err := strconv.Atoi(args)
	if err != nil {
		return usage()
	}

	autoreply, err := s.Channel.Autoreplies(
		models.AutoreplyWhere.Num.EQ(num),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)

	if errors.Is(err, sql.ErrNoRows) {
		return s.Replyf(ctx, "Autoreply #%d does not exist.", num)
	}

	if err != nil {
		return fmt.Errorf("getting autoreply: %w", err)
	}

	if err := autoreply.Delete(ctx, s.Tx); err != nil {
		return fmt.Errorf("deleting autoreply: %w", err)
	}

	return s.Replyf(ctx, "Autoreply #%d has been deleted.", num)
}

func cmdAutoreplyEditResponse(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<index> <response>")
	}

	numStr, response := splitSpace(args)

	if numStr == "" || response == "" {
		return usage()
	}

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return usage()
	}

	var warning string
	if _, malformed := cbp.Parse(response); malformed {
		warning += " Warning: response contains stray (_ or _) separators and may not be processed correctly."
	}

	autoreply, err := s.Channel.Autoreplies(
		models.AutoreplyWhere.Num.EQ(num),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)

	if errors.Is(err, sql.ErrNoRows) {
		return s.Replyf(ctx, "Autoreply #%d does not exist.", num)
	}

	if err != nil {
		return fmt.Errorf("getting autoreply: %w", err)
	}

	autoreply.Response = response
	autoreply.Editor = s.User

	if err := autoreply.Update(ctx, s.Tx, boil.Whitelist(models.AutoreplyColumns.UpdatedAt, models.AutoreplyColumns.Response, models.AutoreplyColumns.Editor)); err != nil {
		return fmt.Errorf("updating autoreply: %w", err)
	}

	return s.Replyf(ctx, "Autoreply #%d's response has been edited.%s", num, warning)
}

func cmdAutoreplyEditPattern(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<index> <pattern>")
	}

	numStr, pattern := splitSpace(args)

	if numStr == "" || pattern == "" {
		return usage()
	}

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return usage()
	}

	trigger, err := s.patternToTrigger(pattern)
	if err != nil {
		return s.replyBadPattern(ctx, err)
	}

	autoreply, err := s.Channel.Autoreplies(
		models.AutoreplyWhere.Num.EQ(num),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)

	if errors.Is(err, sql.ErrNoRows) {
		return s.Replyf(ctx, "Autoreply #%d does not exist.", num)
	}

	if err != nil {
		return fmt.Errorf("getting autoreply: %w", err)
	}

	autoreply.Trigger = trigger
	autoreply.OrigPattern = null.StringFrom(pattern)
	autoreply.Editor = s.User

	if err := autoreply.Update(ctx, s.Tx, boil.Whitelist(models.AutoreplyColumns.UpdatedAt, models.AutoreplyColumns.Trigger, models.AutoreplyColumns.OrigPattern, models.AutoreplyColumns.Editor)); err != nil {
		return fmt.Errorf("updating autoreply: %w", err)
	}

	return s.Replyf(ctx, "Autoreply #%d's pattern has been edited.", num)
}

func cmdAutoreplyList(ctx context.Context, s *session, cmd string, args string) error {
	if !testing.Testing() {
		return s.Replyf(ctx, "You can find the list of autoreplies at: %s/c/%s/autoreplies", s.WebAddr(), s.ChannelName)
	}

	autoreplies, err := s.Channel.Autoreplies(
		qm.OrderBy(models.AutoreplyColumns.Num),
	).All(ctx, s.Tx)
	if err != nil {
		return fmt.Errorf("getting autoreplies: %w", err)
	}

	if len(autoreplies) == 0 {
		return s.Reply(ctx, "There are no autoreplies.")
	}

	var builder strings.Builder
	builder.WriteString("Autoreplies: ")

	for i, autoreply := range autoreplies {
		if i != 0 {
			builder.WriteString(" ; ")
		}

		builder.WriteString(strconv.Itoa(autoreply.Num))
		builder.WriteString(": ")
		builder.WriteString(autoreply.Trigger)
		builder.WriteString(" -> ")
		builder.WriteString(autoreply.Response)
	}

	return s.Reply(ctx, builder.String())
}

const regexPrefix = "REGEX:"

var errEmptyPattern = errors.New("empty pattern")

func (s *session) patternToTrigger(pattern string) (string, error) {
	pattern = strings.ReplaceAll(pattern, "_", " ")

	var trigger string

	if strings.HasPrefix(pattern, regexPrefix) {
		trigger = pattern[len(regexPrefix):]

		if trigger == "" {
			return "", errEmptyPattern
		}
	} else {
		wildcardStart := strings.HasPrefix(pattern, "*")
		wildcardEnd := strings.HasPrefix(pattern, "*")

		pattern = strings.Trim(pattern, "*")

		if pattern == "" {
			return "", errEmptyPattern
		}

		parts := strings.Split(pattern, "*")

		var builder strings.Builder
		builder.WriteByte('^')

		if wildcardStart {
			builder.WriteString(".*")
		}

		for i, p := range parts {
			if p == "" {
				continue
			}

			if i != 0 {
				builder.WriteString(".*")
			}

			// CoeBot did Java's Pattern.quote(), which is equivalent to `\Q` + p + `\E`.
			// Go can quote things without resorting to those special markers, but it's
			// worth noting when looking at older, ported triggers.
			p = regexp.QuoteMeta(p)
			builder.WriteString(p)
		}

		if wildcardEnd {
			builder.WriteString(".*")
		}

		builder.WriteByte('$')

		trigger = builder.String()
	}

	_, err := s.Deps.ReCache.Compile(trigger)
	return trigger, err //nolint:wrapcheck
}

func (s *session) replyBadPattern(ctx context.Context, err error) error {
	var errStr string
	if reErr, ok := errorsx.As[*syntax.Error](err); ok {
		errStr = reErr.Code.String()
	} else {
		errStr = err.Error()
	}

	return s.Replyf(ctx, "Error parsing pattern: %s", errStr)
}

const autoreplyCompactQuery = `
UPDATE autoreplies q
SET num = q3.new_num
FROM (
	SELECT q2.id, q2.num, (ROW_NUMBER() OVER ()) + $2 - 1 AS new_num
	FROM autoreplies q2
	WHERE q2.channel_id = $1 AND q2.num >= $2
	ORDER BY q2.num ASC
) q3
WHERE q3.id = q.id AND q3.id != q3.new_num
`

func cmdAutoreplyCompact(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<num>")
	}

	if args == "" {
		return usage()
	}

	num, err := strconv.Atoi(args)
	if err != nil || num <= 0 {
		return usage()
	}

	result, err := s.Tx.Exec(autoreplyCompactQuery, s.Channel.ID, num)
	if err != nil {
		return fmt.Errorf("compacting autoreplies: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}

	return s.Replyf(ctx, "Compacted autoreplies %d and above (%d affected).", num, affected)
}
