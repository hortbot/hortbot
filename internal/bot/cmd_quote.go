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

var quoteCommands handlerMap = map[string]handlerFunc{
	"add":      {fn: cmdQuoteAdd, minLevel: LevelModerator},
	"delete":   {fn: cmdQuoteDelete, minLevel: LevelModerator},
	"remove":   {fn: cmdQuoteDelete, minLevel: LevelModerator},
	"edit":     {fn: cmdQuoteEdit, minLevel: LevelModerator},
	"getindex": {fn: cmdQuoteGetIndex, minLevel: LevelSubscriber},
	"get":      {fn: cmdQuoteGet, minLevel: LevelSubscriber},
	"random":   {fn: cmdQuoteRandom, minLevel: LevelSubscriber},
	"search":   {fn: cmdQuoteSearch, minLevel: LevelModerator},
	"editor":   {fn: cmdQuoteEditor, minLevel: LevelSubscriber},
}

func cmdQuote(ctx context.Context, s *Session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	if subcommand == "" {
		return cmdQuoteRandom(ctx, s, "", args)
	}

	ok, err := quoteCommands.run(ctx, s, subcommand, args)
	if err != nil {
		return err
	}

	if !ok {
		return cmdQuoteGet(ctx, s, "", subcommand)
	}

	return nil
}

func cmdQuoteAdd(ctx context.Context, s *Session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage("<quote>")
	}

	var row struct {
		MaxNum null.Int
	}

	err := models.NewQuery(
		qm.Select("max("+models.QuoteColumns.Num+") as max_num"),
		qm.From(models.TableNames.Quotes),
		models.QuoteWhere.ChannelID.EQ(s.Channel.ID),
	).Bind(ctx, s.Tx, &row)

	if err != nil {
		return err
	}

	nextNum := row.MaxNum.Int + 1

	quote := &models.Quote{
		ChannelID: s.Channel.ID,
		Num:       nextNum,
		Quote:     args,
		Editor:    s.User,
	}

	if err := quote.Insert(ctx, s.Tx, boil.Infer()); err != nil {
		return err
	}

	return s.Replyf("%s added, this is quote #%d", args, nextNum)
}

func cmdQuoteDelete(ctx context.Context, s *Session, cmd string, args string) error {
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

	quote, err := models.Quotes(
		models.QuoteWhere.ChannelID.EQ(s.Channel.ID),
		models.QuoteWhere.Num.EQ(num),
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return s.Replyf("Quote #%d does not exist.", num)
	}

	if err != nil {
		return err
	}

	if err := quote.Delete(ctx, s.Tx); err != nil {
		return err
	}

	return s.Replyf("Quote #%d has been deleted.", quote.Num)
}

func cmdQuoteEdit(ctx context.Context, s *Session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("<index> <quote>")
	}

	idx, newQuote := splitSpace(args)

	num, err := strconv.Atoi(idx)
	if err != nil {
		return usage()
	}

	if newQuote == "" {
		return usage()
	}

	quote, err := models.Quotes(
		models.QuoteWhere.ChannelID.EQ(s.Channel.ID),
		models.QuoteWhere.Num.EQ(num),
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return s.Replyf("Quote #%d does not exist.", num)
	}

	if err != nil {
		return err
	}

	quote.Quote = newQuote
	quote.Editor = s.User

	if err := quote.Update(ctx, s.Tx, boil.Whitelist(models.QuoteColumns.UpdatedAt, models.QuoteColumns.Quote, models.QuoteColumns.Editor)); err != nil {
		return err
	}

	return s.Replyf("Quote #%d edited.", num)
}

func cmdQuoteGetIndex(ctx context.Context, s *Session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage("<quote>")
	}

	quote, err := models.Quotes(
		models.QuoteWhere.ChannelID.EQ(s.Channel.ID),
		models.QuoteWhere.Quote.EQ(args),
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return s.Reply("Quote not found; make sure your quote is exact.")
	}

	if err != nil {
		return err
	}

	return s.Replyf("That's quote #%d.", quote.Num)
}

func cmdQuoteGet(ctx context.Context, s *Session, cmd string, args string) error {
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

	quote, err := models.Quotes(
		models.QuoteWhere.ChannelID.EQ(s.Channel.ID),
		models.QuoteWhere.Num.EQ(num),
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return s.Replyf("Quote #%d does not exist.", num)
	}

	if err != nil {
		return err
	}

	return s.Replyf("Quote #%d: %s", quote.Num, quote.Quote)
}

func cmdQuoteRandom(ctx context.Context, s *Session, cmd string, args string) error {
	quote, err := models.Quotes(qm.OrderBy("random()")).One(ctx, s.Tx)
	if err == sql.ErrNoRows {
		return s.Reply("there are no quotes")
	}

	if err != nil {
		return err
	}

	return s.Replyf("Quote #%d: %s", quote.Num, quote.Quote)
}

var likeEscaper = strings.NewReplacer(`%`, `\%`, `_`, `\_`)

func cmdQuoteSearch(ctx context.Context, s *Session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage("<phrase>")
	}

	pattern := "%" + likeEscaper.Replace(args) + "%"

	var quotes []struct {
		Num int
	}

	err := models.Quotes(
		qm.Select(models.QuoteColumns.Num),
		qm.Where("quote ILIKE ?", pattern),
	).Bind(ctx, s.Tx, &quotes)

	if err != nil {
		return err
	}

	switch len(quotes) {
	case 0:
		return s.Reply("No quote contained that phrase.")
	case 1:
		return s.Replyf("Phrase found in quote %d", quotes[0].Num)
	}

	var builder strings.Builder
	builder.WriteString("Phrase found in quotes ")

	last := len(quotes) - 1
	for i, q := range quotes {
		builder.WriteString(strconv.Itoa(q.Num))

		if i != last {
			builder.WriteString(", ")
		}
	}

	return s.Reply(builder.String())
}

func cmdQuoteEditor(ctx context.Context, s *Session, cmd string, args string) error {
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

	quote, err := models.Quotes(
		models.QuoteWhere.ChannelID.EQ(s.Channel.ID),
		models.QuoteWhere.Num.EQ(num),
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return s.Replyf("Quote #%d does not exist.", num)
	}

	if err != nil {
		return err
	}

	return s.Replyf("Quote #%d was last edited by %s", quote.Num, quote.Editor)
}
