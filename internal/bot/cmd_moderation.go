package bot

import (
	"context"
	"strconv"
	"time"

	"github.com/hako/durafmt"
	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
)

var moderationCommands = newHandlerMap(map[string]handlerFunc{
	"+b":   {fn: cmdModBan, minLevel: AccessLevelModerator},
	"-b":   {fn: cmdModUnban, minLevel: AccessLevelModerator},
	"+t":   {fn: cmdModTimeout, minLevel: AccessLevelModerator},
	"-t":   {fn: cmdModUntimeout, minLevel: AccessLevelModerator},
	"+p":   {fn: cmdModPurge, minLevel: AccessLevelModerator},
	"+m":   {fn: cmdChangeMode("slow", "Chat is now in slow mode."), minLevel: AccessLevelModerator},
	"-m":   {fn: cmdChangeMode("slowoff", "Chat is no longer in slow mode."), minLevel: AccessLevelModerator},
	"+s":   {fn: cmdChangeMode("subscribers", "Chat is now in subscribers only mode."), minLevel: AccessLevelModerator},
	"-s":   {fn: cmdChangeMode("subscribersoff", "Chat is no longer in subscribers only mode."), minLevel: AccessLevelModerator},
	"+r9k": {fn: cmdChangeMode("r9kbeta", "Chat is now in r9k mode."), minLevel: AccessLevelModerator},
	"-r9k": {fn: cmdChangeMode("r9kbetaoff", "Chat is no longer in r9k mode."), minLevel: AccessLevelModerator},
})

func isModerationCommand(prefix, name string) (prefixAndName string, ok bool) {
	if prefix != "+" && prefix != "-" {
		return "", false
	}

	prefixAndName = prefix + name
	_, ok = moderationCommands.m[prefixAndName]
	return prefixAndName, ok
}

func cmdModBan(ctx context.Context, s *session, cmd string, args string) error {
	user, _ := splitSpace(args)

	if user == "" {
		return s.Reply(ctx, "Usage: +b <user>")
	}

	user = cleanUsername(user)

	reason := "Banned via +b by " + s.UserDisplay
	if err := s.BanByUsername(ctx, user, 0, reason); err != nil {
		if ae, ok := apiclient.AsError(err); ok && ae.IsNotPermitted() {
			return s.Reply(ctx, "Unable to ban user; is the bot modded?")
		}
		return err
	}

	return s.Replyf(ctx, "%s has been banned.", user)
}

func cmdModUnban(ctx context.Context, s *session, cmd string, args string) error {
	user, _ := splitSpace(args)

	if user == "" {
		return s.Reply(ctx, "Usage: -b <user>")
	}

	user = cleanUsername(user)

	if err := s.UnbanByUsername(ctx, user); err != nil {
		if ae, ok := apiclient.AsError(err); ok && ae.IsNotPermitted() {
			return s.Reply(ctx, "Unable to unban user; is the bot modded?")
		}
		return err
	}

	return s.Replyf(ctx, "%s has been unbanned.", user)
}

func cmdModTimeout(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.Reply(ctx, "Usage: +t <user> [seconds]")
	}

	user, args := splitSpace(args)
	seconds, _ := splitSpace(args)

	if user == "" {
		return usage()
	}

	user = cleanUsername(user)
	reason := "Timed out via +t by " + s.UserDisplay

	if seconds == "" {
		if err := s.BanByUsername(ctx, user, 600, reason); err != nil {
			if ae, ok := apiclient.AsError(err); ok && ae.IsNotPermitted() {
				return s.Reply(ctx, "Unable to timeout user; is the bot modded?")
			}
			return err
		}

		return s.Replyf(ctx, "%s has been timed out.", user)
	}

	duration, err := strconv.ParseInt(seconds, 10, 64)
	if err != nil {
		return usage()
	}

	if err := s.BanByUsername(ctx, user, duration, reason); err != nil {
		if ae, ok := apiclient.AsError(err); ok && ae.IsNotPermitted() {
			return s.Reply(ctx, "Unable to timeout user; is the bot modded?")
		}
		return err
	}

	return s.Replyf(ctx, "%s has been timed out for %s seconds.", user, seconds)
}

func cmdModUntimeout(ctx context.Context, s *session, cmd string, args string) error {
	user, _ := splitSpace(args)

	if user == "" {
		return s.Reply(ctx, "Usage: -t <user>")
	}

	user = cleanUsername(user)

	if err := s.UnbanByUsername(ctx, user); err != nil {
		if ae, ok := apiclient.AsError(err); ok && ae.IsNotPermitted() {
			return s.Reply(ctx, "Unable to untimeout user; is the bot modded?")
		}
		return err
	}

	return s.Replyf(ctx, "%s is no longer timed out.", user)
}

func cmdChangeMode(command, message string) func(ctx context.Context, s *session, cmd string, args string) error {
	return func(ctx context.Context, s *session, cmd string, args string) error {
		patch := &twitch.ChatSettingsPatch{}

		switch command {
		case "slow":
			patch.SlowMode = new(true)
		case "slowoff":
			patch.SlowMode = new(false)
		case "subscribers":
			patch.SubscriberMode = new(true)
		case "subscribersoff":
			patch.SubscriberMode = new(false)
		case "r9kbeta":
			patch.UniqueChatMode = new(true)
		case "r9kbetaoff":
			patch.UniqueChatMode = new(false)
		default:
			panic("unknown cmdChangeMode command: " + command)
		}

		if err := s.UpdateChatSettings(ctx, patch); err != nil {
			if ae, ok := apiclient.AsError(err); ok && ae.IsNotPermitted() {
				return s.Reply(ctx, "Unable to update chat settings; is the bot modded?")
			}
			return err
		}
		return s.Reply(ctx, message)
	}
}

func cmdModPurge(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.Reply(ctx, "Usage: +p <user>")
	}

	user, _ := splitSpace(args)

	// TODO: Accept an argument for the last N messages,
	// don't use a timeout and instead remove some large number

	if user == "" {
		return usage()
	}

	user = cleanUsername(user)

	if err := s.BanByUsername(ctx, user, 1, "Purging chat messages"); err != nil {
		return err
	}

	return s.Replyf(ctx, "%s's chat history has been purged.", user)
}

func cmdModClear(ctx context.Context, s *session, cmd string, args string) error {
	return s.ClearChat(ctx)
}

const permitDur = 5 * time.Minute

var permitDurReadable = durafmt.Parse(permitDur).String()

func cmdPermit(ctx context.Context, s *session, cmd string, args string) error {
	if !s.Channel.EnableFilters || !s.Channel.FilterLinks {
		return nil
	}

	user, _ := splitSpace(args)
	if user == "" {
		return s.ReplyUsage(ctx, "<user>")
	}
	user = cleanUsername(user)

	if err := s.LinkPermit(ctx, user, permitDur); err != nil {
		return err
	}

	return s.Replyf(ctx, "%s may now post one link within "+permitDurReadable+".", user)
}
