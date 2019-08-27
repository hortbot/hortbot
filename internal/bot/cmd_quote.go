package bot

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

var quoteCommands = newHandlerMap(map[string]handlerFunc{
	"add":      {fn: cmdQuoteAdd, minLevel: levelModerator},
	"delete":   {fn: cmdQuoteDelete, minLevel: levelModerator},
	"remove":   {fn: cmdQuoteDelete, minLevel: levelModerator},
	"edit":     {fn: cmdQuoteEdit, minLevel: levelModerator},
	"getindex": {fn: cmdQuoteGetIndex, minLevel: levelSubscriber},
	"get":      {fn: cmdQuoteGet, minLevel: levelSubscriber},
	"random":   {fn: cmdQuoteRandom, minLevel: levelSubscriber},
	"search":   {fn: cmdQuoteSearch, minLevel: levelModerator},
	"editor":   {fn: cmdQuoteEditor, minLevel: levelSubscriber},
	"compact":  {fn: cmdQuoteCompact, minLevel: levelSubscriber},
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
		return err
	}

	nextNum := row.MaxNum.Int + 1

	quote := &models.Quote{
		ChannelID: s.Channel.ID,
		Num:       nextNum,
		Quote:     args,
		Creator:   s.User,
		Editor:    s.User,
	}

	if err := quote.Insert(ctx, s.Tx, boil.Infer()); err != nil {
		return err
	}

	return s.Replyf(ctx, "%s added as quote #%d.", args, nextNum)
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

	if err == sql.ErrNoRows {
		return s.Replyf(ctx, "Quote #%d does not exist.", num)
	}

	if err != nil {
		return err
	}

	if err := quote.Delete(ctx, s.Tx); err != nil {
		return err
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

	quote, err := s.Channel.Quotes(
		models.QuoteWhere.Num.EQ(num),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return s.Replyf(ctx, "Quote #%d does not exist.", num)
	}

	if err != nil {
		return err
	}

	quote.Quote = newQuote
	quote.Editor = s.User

	if err := quote.Update(ctx, s.Tx, boil.Whitelist(models.QuoteColumns.UpdatedAt, models.QuoteColumns.Quote, models.QuoteColumns.Editor)); err != nil {
		return err
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

	if err == sql.ErrNoRows {
		return s.Reply(ctx, "Quote not found; make sure your quote is exact.")
	}

	if err != nil {
		return err
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

	if err == sql.ErrNoRows {
		return s.Replyf(ctx, "Quote #%d does not exist.", num)
	}

	if err != nil {
		return err
	}

	return s.Replyf(ctx, "Quote #%d: %s", quote.Num, quote.Quote)
}

func getRandomQuote(ctx context.Context, cx boil.ContextExecutor, channel *models.Channel) (*models.Quote, error) {
	quote, err := channel.Quotes(qm.OrderBy("random()")).One(ctx, cx)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return quote, err
}

func cmdQuoteRandom(ctx context.Context, s *session, cmd string, args string) error {
	quote, err := getRandomQuote(ctx, s.Tx, s.Channel)
	if err != nil {
		return err
	}

	if quote == nil {
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
		qm.Where("quote ILIKE ?", pattern),
	).Bind(ctx, s.Tx, &quotes)

	if err != nil {
		return err
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

	if err == sql.ErrNoRows {
		return s.Replyf(ctx, "Quote #%d does not exist.", num)
	}

	if err != nil {
		return err
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
WHERE q3.id = q.id AND q3.id != q3.new_num
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

	result, err := s.Tx.Exec(quoteCompactQuery, s.Channel.ID, num)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	return s.Replyf(ctx, "Compacted quotes %d and above (%d affected).", num, affected)
}
