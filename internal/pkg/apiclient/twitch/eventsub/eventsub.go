package eventsub

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
)

type UnknownTypeError struct {
	Field string
	Value string
}

func (e *UnknownTypeError) Error() string {
	return fmt.Sprintf("unknown %s: %q", e.Field, e.Value)
}

type WebsocketMessage struct {
	Metadata *WebsocketMessageMetadata `json:"metadata"`
	Payload  any                       `json:"payload"`
}

func (w *WebsocketMessage) UnmarshalJSON(data []byte) error {
	var raw struct {
		Metadata *WebsocketMessageMetadata `json:"metadata"`
		Payload  json.RawMessage           `json:"payload"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	w.Metadata = raw.Metadata

	if unmarshallPayload, ok := payloadFuncs[w.Metadata.MessageType]; ok {
		return unmarshallPayload(raw.Payload, &w.Payload)
	}
	return &UnknownTypeError{Field: "message type", Value: w.Metadata.MessageType}
}

var payloadFuncs = map[string]func([]byte, *any) error{
	"session_welcome":   unmarshallPointerToAny[SessionWelcomePayload],
	"session_keepalive": unmarshallPointerToAny[SessionKeepalivePayload],
	"session_reconnect": unmarshallPointerToAny[SessionReconnectPayload],
	"notification":      unmarshallPointerToAny[NotificationPayload],
}

func unmarshallPointerToAny[T any](data []byte, target *any) error {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*target = &v
	return nil
}

type WebsocketMessageMetadata struct {
	MessageID           string    `json:"message_id"`
	MessageType         string    `json:"message_type"`
	MessageTimestamp    time.Time `json:"message_timestamp"`
	SubscriptionType    string    `json:"subscription_type,omitempty"`
	SubscriptionVersion string    `json:"subscription_version,omitempty"`
}

type Session struct {
	ID                      string    `json:"id"`
	Status                  string    `json:"status"`
	ConnectedAt             time.Time `json:"connected_at"`
	KeepaliveTimeoutSeconds int       `json:"keepalive_timeout_seconds"`
	ReconnectURL            *string   `json:"reconnect_url"`
}

type SessionWelcomePayload struct {
	Session Session `json:"session"`
}

type SessionReconnectPayload struct {
	Session Session `json:"session"`
}

type SessionKeepalivePayload struct{}

type Subscription struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"`
	Type      string     `json:"type"`
	Version   string     `json:"version"`
	Condition any        `json:"condition"`
	Transport *Transport `json:"transport"`
}

func (s *Subscription) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID        string          `json:"id"`
		Status    string          `json:"status"`
		Type      string          `json:"type"`
		Version   string          `json:"version"`
		Condition json.RawMessage `json:"condition"`
		Transport *Transport      `json:"transport"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	s.ID = raw.ID
	s.Status = raw.Status
	s.Type = raw.Type
	s.Version = raw.Version
	s.Transport = raw.Transport

	if unmarshallCondition, ok := subscriptionConditionFuncs[s.Type]; ok {
		return unmarshallCondition(raw.Condition, &s.Condition)
	}
	return &UnknownTypeError{Field: "subscription type", Value: s.Type}
}

type ChatMessageSubscriptionCondition struct {
	BroadcasterUserID idstr.IDStr `json:"broadcaster_user_id"`
	UserID            idstr.IDStr `json:"user_id"`
}

const ChatMessageSubscriptionType = "channel.chat.message"

var subscriptionConditionFuncs = map[string]func([]byte, *any) error{
	ChatMessageSubscriptionType: unmarshallPointerToAny[ChatMessageSubscriptionCondition],
}

type Transport struct {
	// Method is either "websocket", "webhook", or "conduit".
	Method string `json:"method"`

	SessionID string `json:"session_id,omitempty"`

	Callback string `json:"callback,omitempty"`
	Secret   string `json:"secret,omitempty"`

	ConduitID string `json:"conduit_id,omitempty"`
}

type NotificationPayload struct {
	Subscription *Subscription `json:"subscription"`
	Event        any           `json:"event"`
}

func (n *NotificationPayload) UnmarshalJSON(data []byte) error {
	var raw struct {
		Subscription *Subscription   `json:"subscription"`
		Event        json.RawMessage `json:"event"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	n.Subscription = raw.Subscription

	if unmarshallPayload, ok := subscriptionEventFuncs[n.Subscription.Type]; ok {
		return unmarshallPayload(raw.Event, &n.Event)
	}
	return &UnknownTypeError{Field: "subscription type", Value: n.Subscription.Type}
}

var subscriptionEventFuncs = map[string]func([]byte, *any) error{
	ChatMessageSubscriptionType: unmarshallPointerToAny[ChatMessageEvent],
}

type ChatMessageEvent struct {
	BroadcasterUserID           idstr.IDStr             `json:"broadcaster_user_id"`
	BroadcasterUserLogin        string                  `json:"broadcaster_user_login"`
	BroadcasterUserName         string                  `json:"broadcaster_user_name"`
	ChatterUserID               idstr.IDStr             `json:"chatter_user_id"`
	ChatterUserLogin            string                  `json:"chatter_user_login"`
	ChatterUserName             string                  `json:"chatter_user_name"`
	MessageID                   string                  `json:"message_id"`
	Message                     ChatMessageEventMessage `json:"message"`
	MessageType                 string                  `json:"message_type"`
	Badges                      []ChatMessageEventBadge `json:"badges"`
	Cheer                       *ChatMessageEventCheer  `json:"cheer"`
	Color                       string                  `json:"color"`
	Reply                       *ChatMessageEventReply  `json:"reply"`
	ChannelPointsCustomRewardID *string                 `json:"channel_points_custom_reward_id"`
	ChannelPointsAnimationID    *string                 `json:"channel_points_animation_id"`
}

type ChatMessageEventMessage struct {
	Text      string                            `json:"text"`
	Fragments []ChatMessageEventMessageFragment `json:"fragments"`
}

type ChatMessageEventMessageFragment struct {
	Type      string                                    `json:"type"`
	Text      string                                    `json:"text"`
	Cheermote *ChatMessageEventMessageFragmentCheermote `json:"cheermote"`
	Emote     *ChatMessageEventMessageFragmentEmote     `json:"emote"`
	Mention   *ChatMessageEventMessageFragmentMention   `json:"mention"`
}

type ChatMessageEventMessageFragmentCheermote struct {
	Prefix string `json:"prefix"`
	Bits   int    `json:"bits"`
	Tier   int    `json:"tier"`
}

type ChatMessageEventMessageFragmentEmote struct {
	ID         string   `json:"id"`
	EmoteSetID string   `json:"emote_set_id"`
	OwnerID    string   `json:"owner_id"`
	Format     []string `json:"format"`
}

type ChatMessageEventMessageFragmentMention struct {
	UserID    idstr.IDStr `json:"user_id"`
	UserName  string      `json:"user_name"`
	UserLogin string      `json:"user_login"`
}

type ChatMessageEventBadge struct {
	SetID string `json:"set_id"`
	ID    string `json:"id"`
	Info  string `json:"info"`
}

type ChatMessageEventCheer struct {
	Bits int `json:"bits"`
}

type ChatMessageEventReply struct {
	ParentMessageID   string      `json:"parent_message_id"`
	ParentMessageBody string      `json:"parent_message_body"`
	ParentUserID      idstr.IDStr `json:"parent_user_id"`
	ParentUserLogin   string      `json:"parent_user_login"`
	ParentUserName    string      `json:"parent_user_name"`
	ThreadMessageID   string      `json:"thread_message_id"`
	ThreadUserID      idstr.IDStr `json:"thread_user_id"`
	ThreadUserLogin   string      `json:"thread_user_login"`
	ThreadUserName    string      `json:"thread_user_name"`
}
