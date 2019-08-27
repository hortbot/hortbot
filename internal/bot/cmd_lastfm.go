package bot

import (
	"context"
	"strings"
)

func cmdLastFM(ctx context.Context, s *session, cmd string, args string) error {
	if s.Channel.LastFM == "" {
		return errBuiltinDisabled
	}

	if err := s.TryCooldown(); err != nil {
		return err
	}

	return s.Replyf(ctx, "https://www.last.fm/user/%s", s.Channel.LastFM)
}

func cmdMusic(ctx context.Context, s *session, cmd string, args string) error {
	if s.Deps.LastFM == nil || s.Channel.LastFM == "" {
		return errBuiltinDisabled
	}

	if err := s.TryCooldown(); err != nil {
		return err
	}

	resp, err := getSongString(ctx, s, false)
	if err != nil {
		return err
	}

	return s.Reply(ctx, resp)
}

func cmdSonglink(ctx context.Context, s *session, cmd string, args string) error {
	if s.Deps.LastFM == nil || s.Channel.LastFM == "" {
		return errBuiltinDisabled
	}

	if err := s.TryCooldown(); err != nil {
		return err
	}

	resp, err := getSongString(ctx, s, true)
	if err != nil {
		return err
	}

	return s.Reply(ctx, resp)
}

func getSongString(ctx context.Context, s *session, withURL bool) (string, error) {
	tracks, err := s.Tracks(ctx)
	if err != nil {
		// TODO: reply with error message?
		return "", err
	}

	if len(tracks) == 0 {
		return "No songs scrobbled on LastFM.", nil
	}

	track := tracks[0]

	var builder strings.Builder

	if track.NowPlaying {
		builder.WriteString("Currently playing: ")
	} else {
		builder.WriteString("Last played: ")
	}

	builder.WriteString(track.Name)
	builder.WriteString(" by ")
	builder.WriteString(track.Artist)

	if withURL {
		url := track.URL
		if url != "" {
			builder.WriteString(" - ")
			builder.WriteString(url)
		}
	}

	return builder.String(), nil
}
