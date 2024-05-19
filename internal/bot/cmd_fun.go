package bot

import (
	"context"
	"net/url"
	"strconv"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
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
		return s.Reply(ctx, conchResponses[i])
	}

	quote, ok, err := getRandomQuote(ctx, s.Tx, s.Channel)
	if err != nil {
		return err
	}

	if !ok {
		return s.Reply(ctx, "I can provide no help for your situation.")
	}

	return s.Replyf(ctx, "Maybe these words of wisdom can guide you: %s", quote.Quote)
}

func cmdXKCD(ctx context.Context, s *session, cmd string, args string) error {
	if s.Deps.XKCD == nil {
		return errBuiltinDisabled
	}

	if err := s.TryCooldown(ctx); err != nil {
		return err
	}

	id, err := strconv.Atoi(args)
	if err != nil || id <= 0 {
		return s.ReplyUsage(ctx, "<num>")
	}

	c, err := s.Deps.XKCD.GetComic(ctx, id)
	if apiErr, ok := apiclient.AsError(err); ok && apiErr.IsNotFound() {
		return s.Replyf(ctx, "XKCD comic #%d not found.", id)
	}

	if err != nil {
		return err
	}

	return s.Replyf(ctx, "XKCD Comic #%d Title: %s; Image: %s ; Alt-Text: %s", id, c.Title, c.Img, c.Alt)
}

func cmdGoogle(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "<query>")
	}

	link, err := s.ShortenLink(ctx, "https://google.com/search?q="+url.QueryEscape(args))
	if err != nil {
		return err
	}

	return s.Reply(ctx, link)
}

func cmdLink(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "<query>")
	}

	link, err := s.ShortenLink(ctx, "https://lmgtfy.com/?q="+url.QueryEscape(args))
	if err != nil {
		return err
	}
	return s.Replyf(ctx, `Link to "%s": %s`, args, link)
}

func cmdUrban(ctx context.Context, s *session, cmd string, args string) error {
	if s.Deps.Urban == nil || !s.Channel.UrbanEnabled || args == "" {
		return errBuiltinDisabled
	}

	if err := s.TryCooldown(ctx); err != nil {
		return err
	}

	def, err := s.Deps.Urban.Define(ctx, args)
	if err != nil {
		if apiErr, ok := apiclient.AsError(err); ok {
			if apiErr.IsNotFound() {
				return s.Reply(ctx, "Definition not found.")
			}
			if apiErr.IsServerError() {
				return s.Reply(ctx, "An Urban Dictionary server error has occurred.")
			}
		}
		return err
	}

	return s.Replyf(ctx, `"%s"`, def)
}
