package bot

import "context"

func cmdLastFM(ctx context.Context, s *session, cmd string, args string) error {
	if s.Channel.LastFM == "" {
		return errBuiltinDisabled
	}

	return s.Replyf("https://www.last.fm/user/%s", s.Channel.LastFM)
}

func cmdSonglink(ctx context.Context, s *session, cmd string, args string) error {
	if s.Deps.LastFM == nil || s.Channel.LastFM == "" {
		return errBuiltinDisabled
	}

	tracks, err := s.Tracks()
	if err != nil {
		// TODO: reply with error message?
		return err
	}

	if len(tracks) == 0 {
		return s.Reply("No songs scrobbled on LastFM.")
	}

	track := tracks[0]

	if track.NowPlaying {
		return s.Replyf("Currently playing: %s by %s - %s", track.Name, track.Artist, track.URL)
	}

	return s.Replyf("Last played: %s by %s - %s", track.Name, track.Artist, track.URL)
}
