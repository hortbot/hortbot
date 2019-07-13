package bot

import (
	"context"
	"strconv"

	"github.com/hortbot/hortbot/internal/pkg/apis/xkcd"
)

var conchResponses = [...]string{
	"It is certain.",
	"It is decidedly so.",
	"Better not to tell.",
	"You may rely on it.",
	"Don't count on it.",
	"My reply is no.",
	"Very doubtful.",
	"My sources say no.",
	"Most likely.",
	"Signs point to yes.",
	"Outlook doesn't look good.",
	"The future seems hazy on this.",
	"Unable to discern.",
}

func cmdConch(ctx context.Context, s *session, cmd string, args string) error {
	i := s.Deps.Rand.Intn(len(conchResponses) + 1)

	if i < len(conchResponses) {
		return s.Reply(conchResponses[i])
	}

	quote, err := getRandomQuote(ctx, s.Tx, s.Channel)
	if err != nil {
		return err
	}

	if quote == nil {
		return s.Reply("I can provide no help for your situation.")
	}

	return s.Replyf("Maybe these words of wisdom can guide you: %s", quote.Quote)
}

func cmdXKCD(ctx context.Context, s *session, cmd string, args string) error {
	if s.Deps.XKCD == nil {
		return errBuiltinDisabled
	}

	id, err := strconv.Atoi(args)
	if err != nil || id <= 0 {
		return s.ReplyUsage("<num>")
	}

	c, err := s.Deps.XKCD.GetComic(id)

	if err == xkcd.ErrNotFound {
		return s.Replyf("XKCD comic #%d not found.", id)
	}

	if err != nil {
		// TODO: reply with error message?
		return err
	}

	return s.Replyf("XKCD Comic #%d Title: %s; Image: %s ; Alt-Text: %s", id, c.Title, c.Img, c.Alt)
}
