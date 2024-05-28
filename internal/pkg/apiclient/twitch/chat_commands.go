package twitch

import (
	"context"
	"net/http"
	"strconv"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
	"golang.org/x/oauth2"
)

// This file contains API calls which previously were implemented as IRC commands.

type BanRequest struct {
	UserID   idstr.IDStr `json:"user_id"`
	Duration int64       `json:"duration,omitempty"`
	Reason   string      `json:"reason"`
}

// Ban bans a user in a channel as a particular moderator. A duration of zero will cause a permanent ban.
//
// POST https://api.twitch.tv/helix/moderation/bans
func (t *Twitch) Ban(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, ban *BanRequest) (newToken *oauth2.Token, err error) {
	if ban.UserID == 0 || ban.Reason == "" {
		return nil, apiclient.NewStatusError("twitch", http.StatusBadRequest)
	}

	if modToken == nil || modToken.AccessToken == "" {
		return nil, apiclient.NewStatusError("twitch", http.StatusUnauthorized)
	}

	cli := t.clientForUser(ctx, modToken, setToken(&newToken))

	req, err := cli.NewRequest(ctx, helixRoot+"/moderation/bans")
	if err != nil {
		return nil, err
	}
	req.Param("broadcaster_id", strconv.FormatInt(broadcasterID, 10))
	req.Param("moderator_id", strconv.FormatInt(modID, 10))

	body := &struct {
		Data *BanRequest `json:"data"`
	}{
		Data: ban,
	}

	if err := req.BodyJSON(body).Post().Fetch(ctx); err != nil {
		return newToken, apiclient.WrapRequestErr("twitch", err)
	}

	return newToken, nil
}

// Unban unbans a user from a channel.
//
// DELETE https://api.twitch.tv/helix/moderation/bans
func (t *Twitch) Unban(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, userID int64) (newToken *oauth2.Token, err error) {
	if userID == 0 {
		return nil, apiclient.NewStatusError("twitch", http.StatusBadRequest)
	}

	if modToken == nil || modToken.AccessToken == "" {
		return nil, apiclient.NewStatusError("twitch", http.StatusUnauthorized)
	}

	cli := t.clientForUser(ctx, modToken, setToken(&newToken))

	req, err := cli.NewRequest(ctx, helixRoot+"/moderation/bans")
	if err != nil {
		return nil, err
	}
	req.Param("broadcaster_id", strconv.FormatInt(broadcasterID, 10))
	req.Param("moderator_id", strconv.FormatInt(modID, 10))
	req.Param("user_id", strconv.FormatInt(userID, 10))

	if err := req.Delete().Fetch(ctx); err != nil {
		return newToken, apiclient.WrapRequestErr("twitch", err)
	}

	return newToken, nil
}

type ChatSettingsPatch struct {
	EmoteMode *bool `json:"emote_mode,omitempty"`

	FollowerMode         *bool  `json:"follower_mode,omitempty"`
	FollowerModeDuration *int64 `json:"follower_mode_duration,omitempty"`

	NonModeratorChatDelay         *bool  `json:"non_moderator_chat_delay,omitempty"`
	NonModeratorChatDelayDuration *int64 `json:"non_moderator_chat_delay_duration,omitempty"`

	SlowMode         *bool  `json:"slow_mode,omitempty"`
	SlowModeWaitTime *int64 `json:"slow_mode_wait_time,omitempty"`

	SubscriberMode *bool `json:"subscriber_mode,omitempty"`

	UniqueChatMode *bool `json:"unique_chat_mode,omitempty"`
}

// UpdateChatSettings updates the current chat settings.
//
// PATCH https://api.twitch.tv/helix/chat/settings
func (t *Twitch) UpdateChatSettings(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, patch *ChatSettingsPatch) (newToken *oauth2.Token, err error) {
	if patch == nil ||
		(*patch == ChatSettingsPatch{}) ||
		patch.FollowerModeDuration != nil && patch.FollowerMode == nil ||
		patch.NonModeratorChatDelayDuration != nil && patch.NonModeratorChatDelay == nil ||
		patch.SlowModeWaitTime != nil && patch.SlowMode == nil {
		return nil, apiclient.NewStatusError("twitch", http.StatusBadRequest)
	}

	if modToken == nil || modToken.AccessToken == "" {
		return nil, apiclient.NewStatusError("twitch", http.StatusUnauthorized)
	}

	cli := t.clientForUser(ctx, modToken, setToken(&newToken))

	req, err := cli.NewRequest(ctx, helixRoot+"/chat/settings")
	if err != nil {
		return nil, err
	}
	req.Param("broadcaster_id", strconv.FormatInt(broadcasterID, 10))
	req.Param("moderator_id", strconv.FormatInt(modID, 10))

	if err := req.BodyJSON(patch).Patch().Fetch(ctx); err != nil {
		return newToken, apiclient.WrapRequestErr("twitch", err)
	}

	return newToken, nil
}

// SetChatColor sets the chat color for a user.
//
// PUT https://api.twitch.tv/helix/chat/color
func (t *Twitch) SetChatColor(ctx context.Context, userID int64, userToken *oauth2.Token, color string) (newToken *oauth2.Token, err error) {
	if color == "" {
		return nil, apiclient.NewStatusError("twitch", http.StatusBadRequest)
	}

	if userToken == nil || userToken.AccessToken == "" {
		return nil, apiclient.NewStatusError("twitch", http.StatusUnauthorized)
	}

	cli := t.clientForUser(ctx, userToken, setToken(&newToken))

	req, err := cli.NewRequest(ctx, helixRoot+"/chat/color")
	if err != nil {
		return nil, err
	}
	req.Param("user_id", strconv.FormatInt(userID, 10))
	req.Param("color", color)

	if err := req.Put().Fetch(ctx); err != nil {
		return newToken, apiclient.WrapRequestErr("twitch", err)
	}

	return newToken, nil
}

// DeleteChatMessage deletes a message from chat.
//
// DELETE https://api.twitch.tv/helix/moderation/chat
func (t *Twitch) DeleteChatMessage(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, id string) (newToken *oauth2.Token, err error) {
	if id == "" {
		return nil, apiclient.NewStatusError("twitch", http.StatusBadRequest)
	}

	if modToken == nil || modToken.AccessToken == "" {
		return nil, apiclient.NewStatusError("twitch", http.StatusUnauthorized)
	}

	cli := t.clientForUser(ctx, modToken, setToken(&newToken))

	req, err := cli.NewRequest(ctx, helixRoot+"/moderation/chat")
	if err != nil {
		return nil, err
	}
	req.Param("broadcaster_id", strconv.FormatInt(broadcasterID, 10))
	req.Param("moderator_id", strconv.FormatInt(modID, 10))
	req.Param("message_id", id)

	if err := req.Delete().Fetch(ctx); err != nil {
		return newToken, apiclient.WrapRequestErr("twitch", err)
	}

	return newToken, nil
}

// ClearChat deletes all messages in chat.
//
// DELETE https://api.twitch.tv/helix/moderation/chat
func (t *Twitch) ClearChat(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token) (newToken *oauth2.Token, err error) {
	if modToken == nil || modToken.AccessToken == "" {
		return nil, apiclient.NewStatusError("twitch", http.StatusUnauthorized)
	}

	cli := t.clientForUser(ctx, modToken, setToken(&newToken))

	req, err := cli.NewRequest(ctx, helixRoot+"/moderation/chat")
	if err != nil {
		return nil, err
	}
	req.Param("broadcaster_id", strconv.FormatInt(broadcasterID, 10))
	req.Param("moderator_id", strconv.FormatInt(modID, 10))

	if err := req.Delete().Fetch(ctx); err != nil {
		return newToken, apiclient.WrapRequestErr("twitch", err)
	}

	return newToken, nil
}

// Announce makes an announcement in chat.
//
// POST https://api.twitch.tv/helix/chat/announcements
func (t *Twitch) Announce(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, message string, color string) (newToken *oauth2.Token, err error) {
	if message == "" {
		return nil, apiclient.NewStatusError("twitch", http.StatusBadRequest)
	}

	if modToken == nil || modToken.AccessToken == "" {
		return nil, apiclient.NewStatusError("twitch", http.StatusUnauthorized)
	}

	cli := t.clientForUser(ctx, modToken, setToken(&newToken))

	req, err := cli.NewRequest(ctx, helixRoot+"/chat/announcements")
	if err != nil {
		return nil, err
	}
	req.Param("broadcaster_id", strconv.FormatInt(broadcasterID, 10))
	req.Param("moderator_id", strconv.FormatInt(modID, 10))

	body := &struct {
		Message string `json:"message"`
		Color   string `json:"color,omitempty"`
	}{
		Message: message,
		Color:   color,
	}

	if err := req.BodyJSON(body).Post().Fetch(ctx); err != nil {
		return newToken, apiclient.WrapRequestErr("twitch", err)
	}

	return newToken, nil
}

// SendChatMessage sends a chat message as the given user.
//
// POST https://api.twitch.tv/helix/chat/messages
func (t *Twitch) SendChatMessage(ctx context.Context, broadcasterID int64, senderID int64, senderToken *oauth2.Token, message string) (newToken *oauth2.Token, err error) {
	if message == "" {
		return nil, apiclient.NewStatusError("twitch", http.StatusBadRequest)
	}

	if len(message) > 500 {
		message = message[:500]
	}

	if senderToken == nil || senderToken.AccessToken == "" {
		return nil, apiclient.NewStatusError("twitch", http.StatusUnauthorized)
	}

	cli := t.clientForUser(ctx, senderToken, setToken(&newToken))

	req, err := cli.NewRequest(ctx, helixRoot+"/chat/messages")
	if err != nil {
		return nil, err
	}

	body := &struct {
		BroadcasterID idstr.IDStr `json:"broadcaster_id"`
		SenderID      idstr.IDStr `json:"sender_id"`
		Message       string      `json:"message"`
	}{
		BroadcasterID: idstr.IDStr(broadcasterID),
		SenderID:      idstr.IDStr(senderID),
		Message:       message,
	}

	if err := req.BodyJSON(body).Post().Fetch(ctx); err != nil {
		return newToken, apiclient.WrapRequestErr("twitch", err)
	}

	return newToken, nil
}
