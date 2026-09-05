package eventsub

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
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

func unmarshalObject(in *jsontext.Decoder, field func(string, *jsontext.Decoder) error) error {
	start, err := in.ReadToken()
	if err != nil {
		return err //nolint:wrapcheck
	}
	if start.Kind() == jsontext.KindNull {
		return nil
	}
	if start.Kind() != jsontext.KindBeginObject {
		return fmt.Errorf("expected object, got %s", start.Kind())
	}

	for in.PeekKind() != jsontext.KindEndObject {
		name, err := in.ReadToken()
		if err != nil {
			return err //nolint:wrapcheck
		}
		if err := field(name.String(), in); err != nil {
			return err
		}
	}
	_, err = in.ReadToken()
	return err //nolint:wrapcheck
}

type WebsocketMessage struct {
	Metadata *WebsocketMessageMetadata `json:"metadata"`
	Payload  any                       `json:"payload"`
}

func (w *WebsocketMessage) UnmarshalJSONFrom(in *jsontext.Decoder) error {
	var pendingPayload jsontext.Value
	var payload any
	var metadata *WebsocketMessageMetadata

	err := unmarshalObject(in, func(name string, in *jsontext.Decoder) error {
		switch name {
		case "metadata":
			return json.UnmarshalDecode(in, &metadata) //nolint:wrapcheck
		case "payload":
			if metadata == nil {
				var err error
				pendingPayload, err = in.ReadValue()
				pendingPayload = pendingPayload.Clone()
				return err //nolint:wrapcheck
			}
			return unmarshalPayload(in, metadata.MessageType, &payload)
		default:
			return in.SkipValue() //nolint:wrapcheck
		}
	})
	if err != nil {
		return fmt.Errorf("unmarshal websocket message: %w", err)
	}
	if metadata == nil {
		return errors.New("unmarshal websocket message: missing metadata")
	}
	if pendingPayload != nil {
		if err := unmarshalPayload(jsontext.NewDecoder(bytes.NewReader(pendingPayload)), metadata.MessageType, &payload); err != nil {
			return err
		}
	}

	w.Metadata = metadata
	w.Payload = payload
	return nil
}

func unmarshalPayload(in *jsontext.Decoder, messageType string, target *any) error {
	if unmarshal, ok := payloadFuncs[messageType]; ok {
		return unmarshal(in, target)
	}
	return &UnknownTypeError{Field: "message type", Value: messageType}
}

var payloadFuncs = map[string]func(*jsontext.Decoder, *any) error{
	"session_welcome":   unmarshallPointerToAny[SessionWelcomePayload],
	"session_keepalive": unmarshallPointerToAny[SessionKeepalivePayload],
	"session_reconnect": unmarshallPointerToAny[SessionReconnectPayload],
	"notification":      unmarshallPointerToAny[NotificationPayload],
}

func unmarshallPointerToAny[T any](in *jsontext.Decoder, target *any) error {
	var v T
	if err := json.UnmarshalDecode(in, &v); err != nil {
		return fmt.Errorf("unmarshal %T: %w", (*T)(nil), err)
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

func (s *Subscription) UnmarshalJSONFrom(in *jsontext.Decoder) error {
	var pendingCondition jsontext.Value
	var condition any
	var id, status, subscriptionType, version string
	var transport *Transport

	err := unmarshalObject(in, func(name string, in *jsontext.Decoder) error {
		switch name {
		case "id":
			return json.UnmarshalDecode(in, &id) //nolint:wrapcheck
		case "status":
			return json.UnmarshalDecode(in, &status) //nolint:wrapcheck
		case "type":
			return json.UnmarshalDecode(in, &subscriptionType) //nolint:wrapcheck
		case "version":
			return json.UnmarshalDecode(in, &version) //nolint:wrapcheck
		case "condition":
			if subscriptionType == "" {
				var err error
				pendingCondition, err = in.ReadValue()
				pendingCondition = pendingCondition.Clone()
				return err //nolint:wrapcheck
			}
			return unmarshalSubscriptionCondition(in, subscriptionType, &condition)
		case "transport":
			return json.UnmarshalDecode(in, &transport) //nolint:wrapcheck
		default:
			return in.SkipValue() //nolint:wrapcheck
		}
	})
	if err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}
	if pendingCondition != nil {
		if err := unmarshalSubscriptionCondition(jsontext.NewDecoder(bytes.NewReader(pendingCondition)), subscriptionType, &condition); err != nil {
			return err
		}
	}

	s.ID = id
	s.Status = status
	s.Type = subscriptionType
	s.Version = version
	s.Condition = condition
	s.Transport = transport
	return nil
}

type ChatMessageSubscriptionCondition struct {
	BroadcasterUserID idstr.IDStr `json:"broadcaster_user_id"`
	UserID            idstr.IDStr `json:"user_id"`
}

const ChatMessageSubscriptionType = "channel.chat.message"

func unmarshalSubscriptionCondition(in *jsontext.Decoder, subscriptionType string, target *any) error {
	if unmarshal, ok := subscriptionConditionFuncs[subscriptionType]; ok {
		return unmarshal(in, target)
	}
	return &UnknownTypeError{Field: "subscription type", Value: subscriptionType}
}

var subscriptionConditionFuncs = map[string]func(*jsontext.Decoder, *any) error{
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

func (n *NotificationPayload) UnmarshalJSONFrom(in *jsontext.Decoder) error {
	var pendingEvent jsontext.Value
	var event any
	var subscription *Subscription

	err := unmarshalObject(in, func(name string, in *jsontext.Decoder) error {
		switch name {
		case "subscription":
			return json.UnmarshalDecode(in, &subscription) //nolint:wrapcheck
		case "event":
			if subscription == nil {
				var err error
				pendingEvent, err = in.ReadValue()
				pendingEvent = pendingEvent.Clone()
				return err //nolint:wrapcheck
			}
			return unmarshalSubscriptionEvent(in, subscription.Type, &event)
		default:
			return in.SkipValue() //nolint:wrapcheck
		}
	})
	if err != nil {
		return fmt.Errorf("unmarshal notification: %w", err)
	}
	if subscription == nil {
		return errors.New("unmarshal notification: missing subscription")
	}
	if pendingEvent != nil {
		if err := unmarshalSubscriptionEvent(jsontext.NewDecoder(bytes.NewReader(pendingEvent)), subscription.Type, &event); err != nil {
			return err
		}
	}

	n.Subscription = subscription
	n.Event = event
	return nil
}

func unmarshalSubscriptionEvent(in *jsontext.Decoder, subscriptionType string, target *any) error {
	if unmarshal, ok := subscriptionEventFuncs[subscriptionType]; ok {
		return unmarshal(in, target)
	}
	return &UnknownTypeError{Field: "subscription type", Value: subscriptionType}
}

var subscriptionEventFuncs = map[string]func(*jsontext.Decoder, *any) error{
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
