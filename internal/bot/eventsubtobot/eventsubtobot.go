package eventsubtobot

import (
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"github.com/jakebailey/irc"
)

type eventMessage struct {
	origin       string
	metadata     *eventsub.WebsocketMessageMetadata
	subscription *eventsub.Subscription
	condition    *eventsub.ChatMessageSubscriptionCondition
	event        *eventsub.ChatMessageEvent
}

func ToMessage(originMap map[int64]string, m *eventsub.WebsocketMessage) bot.Message {
	if m == nil {
		return nil
	}

	notification := m.Payload.(*eventsub.NotificationPayload)
	subscription := notification.Subscription
	condition := subscription.Condition.(*eventsub.ChatMessageSubscriptionCondition)
	event := notification.Event.(*eventsub.ChatMessageEvent)

	return &eventMessage{
		origin:       originMap[int64(condition.UserID)],
		metadata:     m.Metadata,
		subscription: subscription,
		condition:    condition,
		event:        event,
	}
}

func (m *eventMessage) Origin() string { return m.origin }

func (m *eventMessage) ID() string               { return m.event.MessageID }
func (m *eventMessage) Timestamp() time.Time     { return m.metadata.MessageTimestamp }
func (m *eventMessage) BroadcasterLogin() string { return m.event.BroadcasterUserLogin }
func (m *eventMessage) BroadcasterID() int64     { return int64(m.event.BroadcasterUserID) }
func (m *eventMessage) UserLogin() string        { return m.event.ChatterUserLogin }
func (m *eventMessage) UserDisplay() string      { return m.event.ChatterUserName }
func (m *eventMessage) UserID() int64            { return int64(m.event.ChatterUserID) }

func (m *eventMessage) Message() (message string, me bool) {
	message = m.event.Message.Text

	if c, a, ok := irc.ParseCTCP(message); ok {
		if c != "ACTION" {
			return "", false
		}

		message = a
		me = true
	}

	return strings.TrimSpace(message), me
}

func (m *eventMessage) EmoteCount() int {
	count := 0
	for _, fragment := range m.event.Message.Fragments {
		if fragment.Type == "emote" {
			count++
		}
	}
	return count
}

func (m *eventMessage) UserAccessLevel() bot.AccessLevel {
	if m.BroadcasterID() == m.UserID() {
		return bot.AccessLevelBroadcaster
	}

	badges := make(map[string]bool)

	for _, badge := range m.event.Badges {
		badges[badge.SetID] = true
	}

	switch {
	case badges["broadcaster"]:
		return bot.AccessLevelBroadcaster
	case badges["moderator"]:
		return bot.AccessLevelModerator
	case badges["vip"]:
		return bot.AccessLevelVIP
	case badges["subscriber"], badges["founder"]:
		return bot.AccessLevelSubscriber
	}

	return bot.AccessLevelUnknown
}
