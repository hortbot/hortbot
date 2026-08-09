package conduit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/twitchmocks"
	"gotest.tools/v3/assert"
)

func TestDisabledSubscriptionIsNotCurrent(t *testing.T) {
	t.Parallel()

	sub := &eventsub.Subscription{
		ID:     "revoked",
		Status: "authorization_revoked",
		Type:   eventsub.ChatMessageSubscriptionType,
		Condition: &eventsub.ChatMessageSubscriptionCondition{
			BroadcasterUserID: idstr.IDStr(1),
			UserID:            idstr.IDStr(2),
		},
		Transport: &eventsub.Transport{
			ConduitID: "conduit",
		},
	}

	chatSub := chatSubscription{BroadcasterID: 1, BotID: 2}
	actual, stale, statuses := classifyChatSubscriptions(context.Background(), "conduit", []*eventsub.Subscription{sub})

	assert.DeepEqual(t, actual, map[chatSubscription]string{})
	assert.DeepEqual(t, stale, map[string]chatSubscription{"revoked": chatSub})
	assert.DeepEqual(t, statuses, map[string]int{"authorization_revoked": 1})
}

func TestWebsocketKeepaliveTimeout(t *testing.T) {
	t.Parallel()

	serverErrors := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			serverErrors <- err
			return
		}
		defer conn.CloseNow() //nolint:errcheck

		err = wsjson.Write(r.Context(), conn, &eventsub.WebsocketMessage{
			Metadata: &eventsub.WebsocketMessageMetadata{
				MessageType:      "session_welcome",
				MessageTimestamp: time.Now(),
			},
			Payload: &eventsub.SessionWelcomePayload{
				Session: eventsub.Session{
					ID:                      "session",
					KeepaliveTimeoutSeconds: 1,
				},
			},
		})
		if err != nil {
			serverErrors <- err
			return
		}

		var raw json.RawMessage
		_ = wsjson.Read(r.Context(), conn, &raw)
		serverErrors <- nil
	}))
	defer server.Close()

	api := &twitchmocks.APIMock{
		UpdateShardsFunc: func(context.Context, string, []*twitch.Shard) error {
			return nil
		},
	}
	service := New(nil, api, time.Minute, 1, func(context.Context, json.RawMessage, *eventsub.WebsocketMessage) error {
		return nil
	})
	service.conduitID = "conduit"

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	err := service.runOneWebsocket(ctx, url, 0, nil)
	assert.ErrorContains(t, err, "eventsub keepalive timed out")
	assert.NilError(t, <-serverErrors)
}
