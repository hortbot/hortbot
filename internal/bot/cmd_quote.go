package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/jackc/pgx/v5"
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

	maxNum, err := s.Queries.GetMaxQuoteNumber(ctx, s.Channel.ID)
	if err != nil {
		return fmt.Errorf("getting max quote num: %w", err)
	}

	nextNum := maxNum + 1

	return insertQuote(ctx, s, nextNum, args)
}

func insertQuote(ctx context.Context, s *session, num int32, newQuote string) error {
	err := s.Queries.InsertQuote(ctx, dbsql.InsertQuoteParams{
		ChannelID: s.Channel.ID,
		Num:       num,
		Quote:     newQuote,
		Creator:   s.User,
		Editor:    s.User,
	})
	if err != nil {
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

	num, err := parseInt32(args)
	if err != nil {
		return usage()
	}

	quote, err := s.Queries.GetQuoteByNumberForUpdate(ctx, dbsql.GetQuoteByNumberForUpdateParams{
		ChannelID: s.Channel.ID,
		Num:       num,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return s.Replyf(ctx, "Quote #%d does not exist.", num)
	}

	if err != nil {
		return fmt.Errorf("getting quote: %w", err)
	}

	if err := s.Queries.DeleteQuote(ctx, quote.ID); err != nil {
		return fmt.Errorf("deleting quote: %w", err)
	}

	return s.Replyf(ctx, "Quote #%d has been deleted.", quote.Num)
}

func cmdQuoteEdit(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<index> <quote>")
	}

	idx, newQuote := splitSpace(args)

	num, err := parseInt32(idx)
	if err != nil {
		return usage()
	}

	if newQuote == "" {
		return usage()
	}

	if num <= 0 {
		return s.Reply(ctx, "Quote number cannot be less than one.")
	}

	q := s.Queries
	quote, err := q.GetQuoteByNumberForUpdate(ctx, dbsql.GetQuoteByNumberForUpdateParams{
		ChannelID: s.Channel.ID,
		Num:       num,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		exists, err := q.QuoteExistsAfterNumber(ctx, dbsql.QuoteExistsAfterNumberParams{
			ChannelID: s.Channel.ID,
			Num:       num,
		})
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

	if err := q.UpdateQuote(ctx, dbsql.UpdateQuoteParams{
		Quote:  newQuote,
		Editor: s.User,
		ID:     quote.ID,
	}); err != nil {
		return fmt.Errorf("updating quote: %w", err)
	}

	return s.Replyf(ctx, "Quote #%d edited.", num)
}

func cmdQuoteGetIndex(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "<quote>")
	}

	quote, err := s.Queries.GetQuoteByText(ctx, dbsql.GetQuoteByTextParams{
		ChannelID: s.Channel.ID,
		Quote:     args,
	})

	if errors.Is(err, pgx.ErrNoRows) {
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

	num, err := parseInt32(args)
	if err != nil {
		return usage()
	}

	quote, err := s.Queries.GetQuoteByNumber(ctx, dbsql.GetQuoteByNumberParams{
		ChannelID: s.Channel.ID,
		Num:       num,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return s.Replyf(ctx, "Quote #%d does not exist.", num)
	}

	if err != nil {
		return fmt.Errorf("getting quote: %w", err)
	}

	return s.Replyf(ctx, "Quote #%d: %s", quote.Num, quote.Quote)
}

func getRandomQuote(ctx context.Context, exec *dbsql.Queries, channelID int64) (quote dbsql.Quote, ok bool, err error) {
	quote, err = exec.GetRandomQuote(ctx, channelID)
	if errors.Is(err, pgx.ErrNoRows) {
		return dbsql.Quote{}, false, nil
	}
	if err != nil {
		return dbsql.Quote{}, false, fmt.Errorf("getting random quote: %w", err)
	}
	return quote, true, nil
}

func cmdQuoteRandom(ctx context.Context, s *session, cmd string, args string) error {
	quote, ok, err := getRandomQuote(ctx, s.Queries, s.Channel.ID)
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

	quotes, err := s.Queries.SearchQuoteNumbers(ctx, dbsql.SearchQuoteNumbersParams{
		ChannelID: s.Channel.ID,
		Pattern:   pattern,
	})
	if err != nil {
		return fmt.Errorf("finding quote: %w", err)
	}

	switch len(quotes) {
	case 0:
		return s.Reply(ctx, "No quote contained that phrase.")
	case 1:
		return s.Replyf(ctx, "Phrase found in quote %d.", quotes[0])
	}

	var builder strings.Builder
	builder.WriteString("Phrase found in quotes ")

	last := len(quotes) - 1
	for i, quoteNum := range quotes {
		builder.WriteString(strconv.Itoa(int(quoteNum)))

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

	num, err := parseInt32(args)
	if err != nil {
		return usage()
	}

	quote, err := s.Queries.GetQuoteByNumber(ctx, dbsql.GetQuoteByNumberParams{
		ChannelID: s.Channel.ID,
		Num:       num,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return s.Replyf(ctx, "Quote #%d does not exist.", num)
	}

	if err != nil {
		return fmt.Errorf("getting quote: %w", err)
	}

	return s.Replyf(ctx, "Quote #%d was last edited by %s.", quote.Num, quote.Editor)
}

func cmdQuoteCompact(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<num>")
	}

	if args == "" {
		return usage()
	}

	num, err := parseInt32(args)
	if err != nil || num <= 0 {
		return usage()
	}

	affected, err := s.Queries.CompactQuotes(ctx, dbsql.CompactQuotesParams{
		StartNum:  num,
		ChannelID: s.Channel.ID,
	})
	if err != nil {
		return fmt.Errorf("compacting quotes: %w", err)
	}

	return s.Replyf(ctx, "Compacted quotes %d and above (%d affected).", num, affected)
}
