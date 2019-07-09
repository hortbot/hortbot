package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hortbot/hortbot/internal/cbp"
)

var testingAction func(ctx context.Context, action string) (string, error, bool)

func (s *session) doAction(ctx context.Context, action string) (string, error) {
	if isTesting && testingAction != nil {
		s, err, ok := testingAction(ctx, action)
		if ok {
			return s, err
		}
	}

	// TODO: ORIG_PARAMS to always fetch the entire thing.
	// TODO: Figure out how to deal with change in behavior for PARAMETER (DFS versus BFS)
	// 	     Maybe PARAMETER[0]?

	// TODO: run auto-reply only things first, then check if autoreply and return.

	switch action {
	case "PARAMETER":
		return s.NextParameter(), nil
	case "PARAMETER_CAPS":
		return strings.ToUpper(s.NextParameter()), nil
	case "MESSAGE_COUNT":
		return strconv.FormatInt(s.N, 10), nil
	case "SONG":
		return s.actionSong(0, false)
	case "SONG_URL":
		return s.actionSong(0, true)
	case "LAST_SONG":
		return s.actionSong(1, false)
	}

	return "", fmt.Errorf("unknown action: %s", action)
}

func walk(ctx context.Context, nodes []cbp.Node, fn func(ctx context.Context, action string) (string, error)) (string, error) {
	// Process all commands, converting them to text nodes.
	for i, node := range nodes {
		if node.Text != "" {
			continue
		}

		action, err := walk(ctx, node.Children, fn)
		if err != nil {
			return "", err
		}

		s, err := fn(ctx, action)
		if err != nil {
			return "", err
		}

		nodes[i] = cbp.Node{
			Text: s,
		}
	}

	var sb strings.Builder

	// Merge all strings.
	for _, node := range nodes {
		sb.WriteString(node.Text)
	}

	return sb.String(), nil
}

func (s *session) NextParameter() string {
	var param string
	param, s.CommandParams = splitFirstSep(s.CommandParams, ";")
	return strings.TrimSpace(param)
}

func (s *session) actionSong(i int, url bool) (string, error) {
	// TODO: Precheck commands before running them for simple things (like using SONG without lastfm set).

	tracks, err := s.Tracks()
	if err != nil {
		if err == errLastFMDisabled {
			return "(Unknown)", nil
		}

		return "", err
	}

	if len(tracks) < i+1 {
		return "(Nothing)", nil
	}

	track := tracks[i]

	if url {
		return track.URL, nil
	}

	return track.Name + " by " + track.Artist, nil
}
