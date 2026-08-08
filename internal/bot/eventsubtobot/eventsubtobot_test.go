package eventsubtobot_test

import (
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/bot/eventsubtobot"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
	"gotest.tools/v3/assert"
)

func TestMessage(t *testing.T) {
	t.Parallel()

	msg := toMessage(eventsub.ChatMessageEvent{
		Message: eventsub.ChatMessageEventMessage{Text: "\x01ACTION waves\x01"},
	})

	assert.Equal(t, msg.Text(), "waves")
	assert.Equal(t, msg.IsAction(), true)
}

func TestUserAccessLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userID    int64
		badge     string
		wantLevel bot.AccessLevel
	}{
		{name: "unknown", userID: 2, wantLevel: bot.AccessLevelUnknown},
		{name: "broadcaster", userID: 1, wantLevel: bot.AccessLevelBroadcaster},
		{name: "moderator", userID: 2, badge: "moderator", wantLevel: bot.AccessLevelModerator},
		{name: "lead moderator", userID: 2, badge: "lead_moderator", wantLevel: bot.AccessLevelModerator},
		{name: "VIP", userID: 2, badge: "vip", wantLevel: bot.AccessLevelVIP},
		{name: "subscriber", userID: 2, badge: "subscriber", wantLevel: bot.AccessLevelSubscriber},
		{name: "founder", userID: 2, badge: "founder", wantLevel: bot.AccessLevelSubscriber},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			event := eventsub.ChatMessageEvent{
				BroadcasterUserID: 1,
				ChatterUserID:     idstr.IDStr(test.userID),
			}
			if test.badge != "" {
				event.Badges = []eventsub.ChatMessageEventBadge{{SetID: test.badge}}
			}

			assert.Equal(t, toMessage(event).ChatterAccessLevel(), test.wantLevel)
		})
	}
}

func toMessage(event eventsub.ChatMessageEvent) bot.Message {
	const botID = 999
	sentAt := time.Unix(123, 0)

	return eventsubtobot.ToMessage(map[int64]string{botID: "hortbot"}, &eventsub.WebsocketMessage{
		Metadata: &eventsub.WebsocketMessageMetadata{MessageTimestamp: sentAt},
		Payload: &eventsub.NotificationPayload{
			Subscription: &eventsub.Subscription{
				Condition: &eventsub.ChatMessageSubscriptionCondition{UserID: botID},
			},
			Event: &event,
		},
	})
}
