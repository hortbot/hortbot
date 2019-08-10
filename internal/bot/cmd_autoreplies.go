package bot

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"regexp/syntax"
	"strconv"
	"strings"

	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

var autoreplyCommands = newHandlerMap(map[string]handlerFunc{
	"add":          {fn: cmdAutoreplyAdd, minLevel: levelModerator},
	"delete":       {fn: cmdAutoreplyDelete, minLevel: levelModerator},
	"remove":       {fn: cmdAutoreplyDelete, minLevel: levelModerator},
	"editresponse": {fn: cmdAutoreplyEditResponse, minLevel: levelModerator},
	"editpattern":  {fn: cmdAutoreplyEditPattern, minLevel: levelModerator},
	"edittrigger":  {fn: cmdAutoreplyEditPattern, minLevel: levelModerator},
	"list":         {fn: cmdAutoreplyList, minLevel: levelSubscriber},
})

func cmdAutoreply(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("add|remove|editresponse|editpattern|list")
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
		return s.ReplyUsage("<pattern> <response>")
	}

	trigger, err := s.patternToTrigger(pattern)
	if err != nil {
		return s.replyBadPattern(err)
	}

	if _, err := cbp.Parse(response); err != nil {
		return s.Reply("Error parsing response.")
	}

	var row struct {
		MaxNum null.Int
	}

	err = s.Channel.Autoreplies(
		qm.Select("max("+models.AutoreplyColumns.Num+") as max_num"),
	).Bind(ctx, s.Tx, &row)
	if err != nil {
		return err
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
		return err
	}

	return s.Replyf("Autoreply #%d added.", autoreply.Num)
}

func cmdAutoreplyDelete(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("<index>")
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

	if err == sql.ErrNoRows {
		return s.Replyf("Autoreply #%d does not exist.", num)
	}

	if err != nil {
		return err
	}

	if err := autoreply.Delete(ctx, s.Tx); err != nil {
		return err
	}

	return s.Replyf("Autoreply #%d has been deleted.", num)
}

func cmdAutoreplyEditResponse(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("<index> <response>")
	}

	numStr, response := splitSpace(args)

	if numStr == "" || response == "" {
		return usage()
	}

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return usage()
	}

	if _, err := cbp.Parse(response); err != nil {
		return s.Reply("Error parsing response.")
	}

	autoreply, err := s.Channel.Autoreplies(
		models.AutoreplyWhere.Num.EQ(num),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return s.Replyf("Autoreply #%d does not exist.", num)
	}

	if err != nil {
		return err
	}

	autoreply.Response = response
	autoreply.Editor = s.User

	if err := autoreply.Update(ctx, s.Tx, boil.Whitelist(models.AutoreplyColumns.UpdatedAt, models.AutoreplyColumns.Response, models.AutoreplyColumns.Editor)); err != nil {
		return err
	}

	return s.Replyf("Autoreply #%d's response has been edited.", num)
}

func cmdAutoreplyEditPattern(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("<index> <pattern>")
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
		return s.replyBadPattern(err)
	}

	autoreply, err := s.Channel.Autoreplies(
		models.AutoreplyWhere.Num.EQ(num),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return s.Replyf("Autoreply #%d does not exist.", num)
	}

	if err != nil {
		return err
	}

	autoreply.Trigger = trigger
	autoreply.OrigPattern = null.StringFrom(pattern)
	autoreply.Editor = s.User

	if err := autoreply.Update(ctx, s.Tx, boil.Whitelist(models.AutoreplyColumns.UpdatedAt, models.AutoreplyColumns.Trigger, models.AutoreplyColumns.OrigPattern, models.AutoreplyColumns.Editor)); err != nil {
		return err
	}

	return s.Replyf("Autoreply #%d's pattern has been edited.", num)
}

func cmdAutoreplyList(ctx context.Context, s *session, cmd string, args string) error {
	// TODO: Just link to website?

	autoreplies, err := s.Channel.Autoreplies(
		qm.OrderBy(models.AutoreplyColumns.Num),
	).All(ctx, s.Tx)
	if err != nil {
		return err
	}

	if len(autoreplies) == 0 {
		return s.Reply("There are no autoreplies.")
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

	return s.Reply(builder.String())
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
	return trigger, err
}

func (s *session) replyBadPattern(err error) error {
	var errStr string
	if reErr, ok := err.(*syntax.Error); ok {
		errStr = reErr.Code.String()
	} else {
		errStr = err.Error()
	}

	return s.Replyf("Error parsing pattern: %s", errStr)
}
