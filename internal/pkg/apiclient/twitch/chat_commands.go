package twitch

import (
	"context"
	"net/url"
	"strconv"

	"golang.org/x/oauth2"
)

// This file contains API calls which previously were implemented as IRC commands.

type BanRequest struct {
	UserID   IDStr  `json:"user_id"`
	Duration int64  `json:"duration,omitempty"`
	Reason   string `json:"reason"`
}

// Ban bans a user in a channel as a particular moderator. A duration of zero will cause a permanent ban.
//
// POST https://api.twitch.tv/helix/moderation/bans
func (t *Twitch) Ban(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, req *BanRequest) (newToken *oauth2.Token, err error) {
	if req.UserID == 0 || req.Reason == "" {
		return nil, ErrBadRequest
	}

	if modToken == nil || modToken.AccessToken == "" {
		return nil, ErrNotAuthorized
	}

	cli := t.helixClientForUser(ctx, modToken, setToken(&newToken))
	url := helixRoot +
		"/moderation/bans?broadcaster_id=" + strconv.FormatInt(broadcasterID, 10) +
		"&moderator_id=" + strconv.FormatInt(modID, 10)

	body := &struct {
		Data *BanRequest `json:"data"`
	}{
		Data: req,
	}

	resp, err := cli.Post(ctx, url, body)
	if err != nil {
		return newToken, err
	}
	defer resp.Body.Close()

	return newToken, statusToError(resp.StatusCode)
}

// Unban unbans a user from a channel.
//
// DELETE https://api.twitch.tv/helix/moderation/bans
func (t *Twitch) Unban(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, userID int64) (newToken *oauth2.Token, err error) {
	if userID == 0 {
		return nil, ErrBadRequest
	}

	if modToken == nil || modToken.AccessToken == "" {
		return nil, ErrNotAuthorized
	}

	cli := t.helixClientForUser(ctx, modToken, setToken(&newToken))
	url := helixRoot +
		"/moderation/bans?broadcaster_id=" + strconv.FormatInt(broadcasterID, 10) +
		"&moderator_id=" + strconv.FormatInt(modID, 10) +
		"&user_id=" + strconv.FormatInt(userID, 10)

	resp, err := cli.Delete(ctx, url)
	if err != nil {
		return newToken, err
	}
	defer resp.Body.Close()

	return newToken, statusToError(resp.StatusCode)
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
		return nil, ErrBadRequest
	}

	if modToken == nil || modToken.AccessToken == "" {
		return nil, ErrNotAuthorized
	}

	cli := t.helixClientForUser(ctx, modToken, setToken(&newToken))
	url := helixRoot +
		"/chat/settings?broadcaster_id=" + strconv.FormatInt(broadcasterID, 10) +
		"&moderator_id=" + strconv.FormatInt(modID, 10)

	resp, err := cli.Patch(ctx, url, patch)
	if err != nil {
		return newToken, err
	}
	defer resp.Body.Close()

	return newToken, statusToError(resp.StatusCode)
}

// SetChatColor sets the chat color for a user.
//
// PUT https://api.twitch.tv/helix/chat/color
func (t *Twitch) SetChatColor(ctx context.Context, userID int64, userToken *oauth2.Token, color string) (newToken *oauth2.Token, err error) {
	if color == "" {
		return nil, ErrBadRequest
	}

	if userToken == nil || userToken.AccessToken == "" {
		return nil, ErrNotAuthorized
	}

	cli := t.helixClientForUser(ctx, userToken, setToken(&newToken))
	url := helixRoot + "/chat/color?user_id=" + strconv.FormatInt(userID, 10) + "&color=" + url.QueryEscape(color)

	resp, err := cli.Put(ctx, url, nil)
	if err != nil {
		return newToken, err
	}
	defer resp.Body.Close()

	return newToken, statusToError(resp.StatusCode)
}

// DeleteChatMessage deletes a message from chat.
//
// DELETE https://api.twitch.tv/helix/moderation/chat
func (t *Twitch) DeleteChatMessage(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, id string) (newToken *oauth2.Token, err error) {
	if id == "" {
		return nil, ErrBadRequest
	}

	if modToken == nil || modToken.AccessToken == "" {
		return nil, ErrNotAuthorized
	}

	cli := t.helixClientForUser(ctx, modToken, setToken(&newToken))
	url := helixRoot +
		"/moderation/chat?broadcaster_id=" + strconv.FormatInt(broadcasterID, 10) +
		"&moderator_id=" + strconv.FormatInt(modID, 10) +
		"&message_id=" + url.QueryEscape(id)

	resp, err := cli.Delete(ctx, url)
	if err != nil {
		return newToken, err
	}
	defer resp.Body.Close()

	return newToken, statusToError(resp.StatusCode)
}

// ClearChat deletes all messages in chat.
//
// DELETE https://api.twitch.tv/helix/moderation/chat
func (t *Twitch) ClearChat(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token) (newToken *oauth2.Token, err error) {
	if modToken == nil || modToken.AccessToken == "" {
		return nil, ErrNotAuthorized
	}

	cli := t.helixClientForUser(ctx, modToken, setToken(&newToken))
	url := helixRoot +
		"/moderation/chat?broadcaster_id=" + strconv.FormatInt(broadcasterID, 10) +
		"&moderator_id=" + strconv.FormatInt(modID, 10)

	resp, err := cli.Delete(ctx, url)
	if err != nil {
		return newToken, err
	}
	defer resp.Body.Close()

	return newToken, statusToError(resp.StatusCode)
}

// Announce makes an announcement in chat.
//
// POST https://api.twitch.tv/helix/chat/announcements
func (t *Twitch) Announce(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, message string, color string) (newToken *oauth2.Token, err error) {
	if message == "" {
		return nil, ErrBadRequest
	}

	if modToken == nil || modToken.AccessToken == "" {
		return nil, ErrNotAuthorized
	}

	cli := t.helixClientForUser(ctx, modToken, setToken(&newToken))
	url := helixRoot +
		"/chat/announcements?broadcaster_id=" + strconv.FormatInt(broadcasterID, 10) +
		"&moderator_id=" + strconv.FormatInt(modID, 10)

	body := &struct {
		Message string `json:"message"`
		Color   string `json:"color,omitempty"`
	}{
		Message: message,
		Color:   color,
	}

	resp, err := cli.Post(ctx, url, body)
	if err != nil {
		return newToken, err
	}
	defer resp.Body.Close()

	return newToken, statusToError(resp.StatusCode)
}
