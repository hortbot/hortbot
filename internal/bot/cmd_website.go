package bot

import "context"

func cmdCommands(ctx context.Context, s *session, cmd string, args string) error {
	return s.Replyf(ctx, "You can find the list of commands at: %s/c/%s/commands", s.WebAddr(), s.ChannelName)
}

func cmdQuotes(ctx context.Context, s *session, cmd string, args string) error {
	return s.Replyf(ctx, "You can find the list of quotes at: %s/c/%s/quotes", s.WebAddr(), s.ChannelName)
}

func cmdHelp(ctx context.Context, s *session, cmd string, args string) error {
	return s.Reply(ctx, s.HelpMessage())
}
