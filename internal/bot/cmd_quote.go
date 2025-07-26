package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/hortbot/hortbot/internal/db/models"
)

var quoteCommands = newHandlerMap(map[string]handlerFunc{
	"add":      {fn: cmdQuoteAdd, minLevel: AccessLevelModerator},
	"delete":   {fn: cmdQuoteDelete, minLevel: AccessLevelModerator},
	"remove":   {fn: cmdQuoteDelete, minLevel: AccessLevelModerator},
	"edit":     {fn: cmdQuoteEdit, minLevel: AccessLevelModerator},
	"getindex": {fn: cmdQuoteGetIndex, minLevel: AccessLevelSubscriber},
	"get":      {fn: cmdQuoteGet, minLevel: AccessLevelSubscriber},
	"random":   {fn: cmdQuoteRandom, minLevel: AccessLevelSubscriber},
	"search":   {fn: cmdQuoteSearch, minLevel: AccessLevelModerator},
	"editor":   {fn: cmdQuoteEditor, minLevel: AccessLevelSubscriber},
	"compact":  {fn: cmdQuoteCompact, minLevel: AccessLevelModerator},
})

func cmdQuote(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	if subcommand == "" {
		return cmdQuoteRandom(ctx, s, "", args)
	}

	ok, err := quoteCommands.Run(ctx, s, subcommand, args)
	if err != nil {
		return err
	}

	if !ok {
		return cmdQuoteGet(ctx, s, "", subcommand)
	}

	return nil
}

func cmdQuoteAdd(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "<quote>")
	}

	var row struct {
		MaxNum null.Int
	}

	err := s.Channel.Quotes(
		qm.Select("max("+models.QuoteColumns.Num+") as max_num"),
	).Bind(ctx, s.Tx, &row)
	if err != nil {
		return fmt.Errorf("getting max quote num: %w", err)
	}

	nextNum := row.MaxNum.Int + 1

	return insertQuote(ctx, s, nextNum, args)
}

func insertQuote(ctx context.Context, s *session, num int, newQuote string) error {
	quote := &models.Quote{
		ChannelID: s.Channel.ID,
		Num:       num,
		Quote:     newQuote,
		Creator:   s.User,
		Editor:    s.User,
	}

	if err := quote.Insert(ctx, s.Tx, boil.Infer()); err != nil {
		return fmt.Errorf("inserting quote: %w", err)
	}

	return s.Replyf(ctx, "%s added as quote #%d.", newQuote, num)
}

func cmdQuoteDelete(ctx context.Context, s *session, cmd string, args string) error {
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

	quote, err := s.Channel.Quotes(
		models.QuoteWhere.Num.EQ(num),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)

	if errors.Is(err, sql.ErrNoRows) {
		return s.Replyf(ctx, "Quote #%d does not exist.", num)
	}

	if err != nil {
		return fmt.Errorf("getting quote: %w", err)
	}

	if err := quote.Delete(ctx, s.Tx); err != nil {
		return fmt.Errorf("deleting quote: %w", err)
	}

	return s.Replyf(ctx, "Quote #%d has been deleted.", quote.Num)
}

func cmdQuoteEdit(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<index> <quote>")
	}

	idx, newQuote := splitSpace(args)

	num, err := strconv.Atoi(idx)
	if err != nil {
		return usage()
	}

	if newQuote == "" {
		return usage()
	}

	if num <= 0 {
		return s.Reply(ctx, "Quote number cannot be less than one.")
	}

	quote, err := s.Channel.Quotes(
		models.QuoteWhere.Num.EQ(num),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)

	if errors.Is(err, sql.ErrNoRows) {
		exists, err := s.Channel.Quotes(models.QuoteWhere.Num.GT(num)).Exists(ctx, s.Tx)
		if err != nil {
			return fmt.Errorf("checking for quotes after index: %w", err)
		}

		// No quotes after the index; don't allow arbitrary edits.
		if !exists {
			return s.Replyf(ctx, "Quote #%d does not exist.", num)
		}

		// Editing a missing quote, insert one.
		return insertQuote(ctx, s, num, newQuote)
	}

	if err != nil {
		return fmt.Errorf("getting quote: %w", err)
	}

	quote.Quote = newQuote
	quote.Editor = s.User

	if err := quote.Update(ctx, s.Tx, boil.Whitelist(models.QuoteColumns.UpdatedAt, models.QuoteColumns.Quote, models.QuoteColumns.Editor)); err != nil {
		return fmt.Errorf("updating quote: %w", err)
	}

	return s.Replyf(ctx, "Quote #%d edited.", num)
}

func cmdQuoteGetIndex(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "<quote>")
	}

	quote, err := s.Channel.Quotes(
		models.QuoteWhere.Quote.EQ(args),
	).One(ctx, s.Tx)

	if errors.Is(err, sql.ErrNoRows) {
		return s.Reply(ctx, "Quote not found; make sure your quote is exact.")
	}

	if err != nil {
		return fmt.Errorf("getting quote: %w", err)
	}

	return s.Replyf(ctx, "That's quote #%d.", quote.Num)
}

func cmdQuoteGet(ctx context.Context, s *session, cmd string, args string) error {
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

	quote, err := s.Channel.Quotes(
		models.QuoteWhere.Num.EQ(num),
	).One(ctx, s.Tx)

	if errors.Is(err, sql.ErrNoRows) {
		return s.Replyf(ctx, "Quote #%d does not exist.", num)
	}

	if err != nil {
		return fmt.Errorf("getting quote: %w", err)
	}

	return s.Replyf(ctx, "Quote #%d: %s", quote.Num, quote.Quote)
}

func getRandomQuote(ctx context.Context, cx boil.ContextExecutor, channel *models.Channel) (quote *models.Quote, ok bool, err error) {
	quote, err = channel.Quotes(qm.OrderBy("random()")).One(ctx, cx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("getting random quote: %w", err)
	}
	return quote, true, nil
}

func cmdQuoteRandom(ctx context.Context, s *session, cmd string, args string) error {
	quote, ok, err := getRandomQuote(ctx, s.Tx, s.Channel)
	if err != nil {
		return err
	}

	if !ok {
		return s.Reply(ctx, "There are no quotes.")
	}

	return s.Replyf(ctx, "Quote #%d: %s", quote.Num, quote.Quote)
}

var likeEscaper = strings.NewReplacer(`%`, `\%`, `_`, `\_`)

func cmdQuoteSearch(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "<phrase>")
	}

	pattern := "%" + likeEscaper.Replace(args) + "%"

	var quotes []struct {
		Num int
	}

	err := s.Channel.Quotes(
		qm.Select(models.QuoteColumns.Num),
		models.QuoteWhere.Quote.ILIKE(pattern),
	).Bind(ctx, s.Tx, &quotes)
	if err != nil {
		return fmt.Errorf("finding quote: %w", err)
	}

	switch len(quotes) {
	case 0:
		return s.Reply(ctx, "No quote contained that phrase.")
	case 1:
		return s.Replyf(ctx, "Phrase found in quote %d.", quotes[0].Num)
	}

	var builder strings.Builder
	builder.WriteString("Phrase found in quotes ")

	last := len(quotes) - 1
	for i, q := range quotes {
		builder.WriteString(strconv.Itoa(q.Num))

		switch {
		case i == last-1:
			if len(quotes) != 2 {
				builder.WriteByte(',')
			}
			builder.WriteString(" and ")
		case i != last:
			builder.WriteString(", ")
		}
	}

	builder.WriteByte('.')

	return s.Reply(ctx, builder.String())
}

func cmdQuoteEditor(ctx context.Context, s *session, cmd string, args string) error {
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

	quote, err := s.Channel.Quotes(
		models.QuoteWhere.Num.EQ(num),
	).One(ctx, s.Tx)

	if errors.Is(err, sql.ErrNoRows) {
		return s.Replyf(ctx, "Quote #%d does not exist.", num)
	}

	if err != nil {
		return fmt.Errorf("getting quote: %w", err)
	}

	return s.Replyf(ctx, "Quote #%d was last edited by %s.", quote.Num, quote.Editor)
}

const quoteCompactQuery = `
UPDATE quotes q
SET num = q3.new_num
FROM (
	SELECT q2.id, q2.num, (ROW_NUMBER() OVER ()) + $2 - 1 AS new_num
	FROM quotes q2
	WHERE q2.channel_id = $1 AND q2.num >= $2
	ORDER BY q2.num ASC
) q3
WHERE q3.id = q.id AND q3.num != q3.new_num
`

func cmdQuoteCompact(ctx context.Context, s *session, cmd string, args string) error {
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

	result, err := s.Tx.ExecContext(ctx, quoteCompactQuery, s.Channel.ID, num)
	if err != nil {
		return fmt.Errorf("compacting quotes: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}

	return s.Replyf(ctx, "Compacted quotes %d and above (%d affected).", num, affected)
}
