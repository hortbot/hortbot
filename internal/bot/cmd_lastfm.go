package bot

import (
	"context"
	"strings"
)

func cmdLastFM(ctx context.Context, s *session, cmd string, args string) error {
	if s.Channel.LastFM == "" {
		return errBuiltinDisabled
	}

	return s.Replyf("https://www.last.fm/user/%s", s.Channel.LastFM)
}

func cmdMusic(ctx context.Context, s *session, cmd string, args string) error {
	if s.Deps.LastFM == nil || s.Channel.LastFM == "" {
		return errBuiltinDisabled
	}

	resp, err := getSongString(s, false)
	if err != nil {
		return err
	}

	return s.Reply(resp)
}

func cmdSonglink(ctx context.Context, s *session, cmd string, args string) error {
	if s.Deps.LastFM == nil || s.Channel.LastFM == "" {
		return errBuiltinDisabled
	}

	resp, err := getSongString(s, true)
	if err != nil {
		return err
	}

	return s.Reply(resp)
}

func getSongString(s *session, withURL bool) (string, error) {
	tracks, err := s.Tracks()
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
