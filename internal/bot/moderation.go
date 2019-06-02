package bot

import (
	"context"
	"strconv"
	"strings"
)

var moderationCommands builtinMap = map[string]builtinCommand{
	"+b":   {fn: cmdModBan, minLevel: LevelModerator},
	"-b":   {fn: cmdModUnban, minLevel: LevelModerator},
	"+t":   {fn: cmdModTimeout, minLevel: LevelModerator},
	"-t":   {fn: cmdModUntimeout, minLevel: LevelModerator},
	"+p":   {fn: cmdModPurge, minLevel: LevelModerator},
	"+m":   {fn: cmdChangeMode("slow", "chat is now in slow mode"), minLevel: LevelModerator},
	"-m":   {fn: cmdChangeMode("slowoff", "chat is no longer in slow mode"), minLevel: LevelModerator},
	"+s":   {fn: cmdChangeMode("subscribers", "chat is now in subscribers only mode"), minLevel: LevelModerator},
	"-s":   {fn: cmdChangeMode("subscribersoff", "chat is no longer in subscribers only mode"), minLevel: LevelModerator},
	"+r9k": {fn: cmdChangeMode("r9kbeta", "chat is now in r9k mode"), minLevel: LevelModerator},
	"-r9k": {fn: cmdChangeMode("r9kbetaoff", "chat is no longer in r9k mode"), minLevel: LevelModerator},
}

func cmdModBan(ctx context.Context, s *Session, cmd string, args string) error {
	user, _ := splitSpace(args)

	if user == "" {
		return s.Reply("usage: +b <user>")
	}

	user = strings.ToLower(user)

	if err := s.SendCommand("ban", user); err != nil {
		return err
	}

	return s.Replyf("%s has been banned", user)
}

func cmdModUnban(ctx context.Context, s *Session, cmd string, args string) error {
	user, _ := splitSpace(args)

	if user == "" {
		return s.Reply("usage: -b <user>")
	}

	user = strings.ToLower(user)

	if err := s.SendCommand("unban", user); err != nil {
		return err
	}

	return s.Replyf("%s has been unbanned", user)
}

func cmdModTimeout(ctx context.Context, s *Session, cmd string, args string) error {
	usage := func() error {
		return s.Reply("usage: +t <user> [seconds]")
	}

	user, args := splitSpace(args)
	seconds, _ := splitSpace(args)

	if user == "" {
		return usage()
	}

	user = strings.ToLower(user)

	if seconds == "" {
		if err := s.SendCommand("timeout", user); err != nil {
			return err
		}

		return s.Replyf("%s has been timed out", user)
	}

	if _, err := strconv.Atoi(seconds); err != nil {
		return usage()
	}

	if err := s.SendCommand("timeout", user, seconds); err != nil {
		return err
	}

	return s.Replyf("%s has been timed out for %s seconds", user, seconds)
}

func cmdModUntimeout(ctx context.Context, s *Session, cmd string, args string) error {
	user, _ := splitSpace(args)

	if user == "" {
		return s.Reply("usage: -t <user>")
	}

	user = strings.ToLower(user)

	if err := s.SendCommand("untimeout", user); err != nil {
		return err
	}

	return s.Replyf("%s is no longer timed out", user)
}

func cmdChangeMode(command, message string) func(ctx context.Context, s *Session, cmd string, args string) error {
	return func(ctx context.Context, s *Session, cmd string, args string) error {
		if err := s.SendCommand(command); err != nil {
			return err
		}
		return s.Reply(message)
	}
}

func cmdModPurge(ctx context.Context, s *Session, cmd string, args string) error {
	usage := func() error {
		return s.Reply("usage: +p <user>")
	}

	user, _ := splitSpace(args)

	// TODO: Accept an argument for the last N messages,
	// don't use a timeout and instead remove some large number

	if user == "" {
		return usage()
	}

	user = strings.ToLower(user)

	if err := s.SendCommand("timeout", user, "1"); err != nil {
		return err
	}

	return s.Replyf("%s's chat history has been purged", user)
}
