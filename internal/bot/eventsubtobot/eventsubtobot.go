package eventsubtobot

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
)

type chatMessage struct {
	botLogin string
	sentAt   time.Time
	text     string
	isAction bool
	event    *eventsub.ChatMessageEvent
	raw      *eventsub.WebsocketMessage
}

func ToMessage(botLoginMap map[int64]string, m *eventsub.WebsocketMessage) bot.Message {
	if m == nil {
		return nil
	}

	notification := m.Payload.(*eventsub.NotificationPayload)
	subscription := notification.Subscription
	condition := subscription.Condition.(*eventsub.ChatMessageSubscriptionCondition)
	event := notification.Event.(*eventsub.ChatMessageEvent)

	text, isAction := parseMessageText(event.Message.Text)

	return &chatMessage{
		botLogin: botLoginMap[int64(condition.UserID)],
		sentAt:   m.Metadata.MessageTimestamp,
		text:     text,
		isAction: isAction,
		event:    event,
		raw:      m,
	}
}

func (m *chatMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		BotLogin string                     `json:"bot_login"`
		Message  *eventsub.WebsocketMessage `json:"message"`
	}{
		BotLogin: m.botLogin,
		Message:  m.raw,
	}) //nolint:wrapcheck
}

func (m *chatMessage) Bot() string                 { return m.botLogin }
func (m *chatMessage) MessageID() string           { return m.event.MessageID }
func (m *chatMessage) MessageTimestamp() time.Time { return m.sentAt }
func (m *chatMessage) Text() string                { return m.text }
func (m *chatMessage) IsAction() bool              { return m.isAction }

func (m *chatMessage) Broadcaster() bot.ChatIdentity {
	return bot.ChatIdentity{
		ID:          int64(m.event.BroadcasterUserID),
		Login:       m.event.BroadcasterUserLogin,
		DisplayName: m.event.BroadcasterUserName,
	}
}

func (m *chatMessage) Chatter() bot.ChatIdentity {
	return bot.ChatIdentity{
		ID:          int64(m.event.ChatterUserID),
		Login:       m.event.ChatterUserLogin,
		DisplayName: m.event.ChatterUserName,
	}
}

func (m *chatMessage) CountEmotes() int {
	count := 0
	for _, fragment := range m.event.Message.Fragments {
		if fragment.Type == "emote" {
			count++
		}
	}
	return count
}

func (m *chatMessage) ChatterAccessLevel() bot.AccessLevel {
	return accessLevel(m.event)
}

func parseMessageText(message string) (text string, isAction bool) {
	if len(message) >= 2 && message[0] == '\x01' && message[len(message)-1] == '\x01' {
		command, args, _ := strings.Cut(message[1:len(message)-1], " ")
		if command != "ACTION" {
			return "", false
		}
		return strings.TrimSpace(args), true
	}
	return strings.TrimSpace(message), false
}

func accessLevel(event *eventsub.ChatMessageEvent) bot.AccessLevel {
	if event.BroadcasterUserID == event.ChatterUserID {
		return bot.AccessLevelBroadcaster
	}

	badges := make(map[string]bool)

	for _, badge := range event.Badges {
		badges[badge.SetID] = true
	}

	switch {
	case badges["broadcaster"]:
		return bot.AccessLevelBroadcaster
	case badges["moderator"], badges["lead_moderator"]:
		return bot.AccessLevelModerator
	case badges["vip"]:
		return bot.AccessLevelVIP
	case badges["subscriber"], badges["founder"]:
		return bot.AccessLevelSubscriber
	}

	return bot.AccessLevelUnknown
}
