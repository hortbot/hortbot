package twitch_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

func TestBan(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const broadcasterID = 1
	const modID = 123
	tok := tokFor(ctx, t, tw, ft, modID)

	newToken, err := tw.Ban(ctx, broadcasterID, modID, tok, &twitch.BanRequest{
		UserID:   666,
		Duration: 30,
		Reason:   "Broke a rule.",
	})

	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)
}

func TestBanBadParameters(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const broadcasterID = 1
	const modID = 123
	tok := tokFor(ctx, t, tw, ft, modID)

	_, err := tw.Ban(ctx, broadcasterID, modID, tok, &twitch.BanRequest{
		UserID:   0,
		Duration: 30,
		Reason:   "Broke a rule.",
	})

	assert.ErrorIs(t, err, twitch.ErrBadRequest)

	_, err = tw.Ban(ctx, broadcasterID, modID, tok, &twitch.BanRequest{
		UserID:   666,
		Duration: 30,
		Reason:   "",
	})

	assert.ErrorIs(t, err, twitch.ErrBadRequest)

	_, err = tw.Ban(ctx, broadcasterID, modID, nil, &twitch.BanRequest{
		UserID:   666,
		Duration: 30,
		Reason:   "Broke a rule.",
	})

	assert.ErrorIs(t, err, twitch.ErrNotAuthorized)

	_, err = tw.Ban(ctx, broadcasterID, modID, &oauth2.Token{}, &twitch.BanRequest{
		UserID:   666,
		Duration: 30,
		Reason:   "Broke a rule.",
	})

	assert.ErrorIs(t, err, twitch.ErrNotAuthorized)
}

func TestBanErrors(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const broadcasterID = 1
	const modID = 123
	tok := tokFor(ctx, t, tw, ft, modID)

	banRequest := &twitch.BanRequest{
		UserID:   666,
		Duration: 30,
		Reason:   "Broke a rule.",
	}

	_, err := tw.Ban(ctx, 777, modID, tok, banRequest)
	assert.ErrorContains(t, err, errTestBadRequest.Error())

	for status, expected := range expectedErrors {
		id := int64(status)
		tok := tokFor(ctx, t, tw, ft, id)

		newToken, err := tw.Ban(ctx, id, modID, tok, banRequest)
		assert.Equal(t, err, expected, "%d", status)
		assert.Assert(t, newToken == nil)
	}
}

func TestUnban(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const broadcasterID = 1234
	const modID = 3141
	tok := tokFor(ctx, t, tw, ft, modID)

	newToken, err := tw.Unban(ctx, broadcasterID, modID, tok, 666)

	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)
}

func TestUnbanBadParameters(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const broadcasterID = 1234
	const modID = 3141
	tok := tokFor(ctx, t, tw, ft, modID)

	_, err := tw.Unban(ctx, broadcasterID, modID, tok, 0)
	assert.ErrorIs(t, err, twitch.ErrBadRequest)

	_, err = tw.Unban(ctx, broadcasterID, modID, nil, 666)
	assert.ErrorIs(t, err, twitch.ErrNotAuthorized)

	_, err = tw.Unban(ctx, broadcasterID, modID, &oauth2.Token{}, 666)
	assert.ErrorIs(t, err, twitch.ErrNotAuthorized)
}

func TestUnbanErrors(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const modID = 3141
	tok := tokFor(ctx, t, tw, ft, modID)

	_, err := tw.Unban(ctx, 777, modID, tok, 666)
	assert.ErrorContains(t, err, errTestBadRequest.Error())

	for status, expected := range expectedErrors {
		id := int64(status)
		tok := tokFor(ctx, t, tw, ft, id)

		newToken, err := tw.Unban(ctx, id, modID, tok, 666)
		assert.Equal(t, err, expected, "%d", status)
		assert.Assert(t, newToken == nil)
	}
}

func TestSetChatColor(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const userID = 1234
	tok := tokFor(ctx, t, tw, ft, userID)

	newToken, err := tw.SetChatColor(ctx, userID, tok, "#9146FF")

	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)
}

func TestSetChatColorBadParameters(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const userID = 1234
	tok := tokFor(ctx, t, tw, ft, userID)

	_, err := tw.SetChatColor(ctx, userID, tok, "")
	assert.ErrorIs(t, err, twitch.ErrBadRequest)

	_, err = tw.SetChatColor(ctx, userID, nil, "#9146FF")
	assert.ErrorIs(t, err, twitch.ErrNotAuthorized)

	_, err = tw.SetChatColor(ctx, userID, &oauth2.Token{}, "#9146FF")
	assert.ErrorIs(t, err, twitch.ErrNotAuthorized)
}

func TestSetChatColorErrors(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	tok := tokFor(ctx, t, tw, ft, 777)

	_, err := tw.SetChatColor(ctx, 777, tok, "#9146FF")
	assert.ErrorContains(t, err, errTestBadRequest.Error())

	for status, expected := range expectedErrors {
		id := int64(status)
		tok := tokFor(ctx, t, tw, ft, id)

		newToken, err := tw.SetChatColor(ctx, id, tok, "#9146FF")
		assert.Equal(t, err, expected, "%d", status)
		assert.Assert(t, newToken == nil)
	}
}

func TestDeleteChatMessage(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const broadcasterID = 1234
	const modID = 3141
	tok := tokFor(ctx, t, tw, ft, modID)

	newToken, err := tw.DeleteChatMessage(ctx, broadcasterID, modID, tok, "somemessage")

	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)
}

func TestDeleteChatMessageBadParameters(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const broadcasterID = 1234
	const modID = 3141
	tok := tokFor(ctx, t, tw, ft, modID)

	_, err := tw.DeleteChatMessage(ctx, broadcasterID, modID, tok, "")
	assert.ErrorIs(t, err, twitch.ErrBadRequest)

	_, err = tw.DeleteChatMessage(ctx, broadcasterID, modID, nil, "somemessage")
	assert.ErrorIs(t, err, twitch.ErrNotAuthorized)

	_, err = tw.DeleteChatMessage(ctx, broadcasterID, modID, &oauth2.Token{}, "somemessage")
	assert.ErrorIs(t, err, twitch.ErrNotAuthorized)
}

func TestDeleteChatMessageErrors(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const modID = 3141
	tok := tokFor(ctx, t, tw, ft, modID)

	_, err := tw.DeleteChatMessage(ctx, 777, modID, tok, "somemessage")
	assert.ErrorContains(t, err, errTestBadRequest.Error())

	for status, expected := range expectedErrors {
		id := int64(status)
		tok := tokFor(ctx, t, tw, ft, id)

		newToken, err := tw.DeleteChatMessage(ctx, id, modID, tok, "somemessage")
		assert.Equal(t, err, expected, "%d", status)
		assert.Assert(t, newToken == nil)
	}
}

func TestClearChat(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const broadcasterID = 1234
	const modID = 3141
	tok := tokFor(ctx, t, tw, ft, modID)

	newToken, err := tw.ClearChat(ctx, broadcasterID, modID, tok)

	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)
}

func TestClearChatBadParameters(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const broadcasterID = 1234
	const modID = 3141

	_, err := tw.ClearChat(ctx, broadcasterID, modID, nil)
	assert.ErrorIs(t, err, twitch.ErrNotAuthorized)

	_, err = tw.ClearChat(ctx, broadcasterID, modID, &oauth2.Token{})
	assert.ErrorIs(t, err, twitch.ErrNotAuthorized)
}

func TestClearChatErrors(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const modID = 3141
	tok := tokFor(ctx, t, tw, ft, modID)

	_, err := tw.ClearChat(ctx, 777, modID, tok)
	assert.ErrorContains(t, err, errTestBadRequest.Error())

	for status, expected := range expectedErrors {
		id := int64(status)
		tok := tokFor(ctx, t, tw, ft, id)

		newToken, err := tw.ClearChat(ctx, id, modID, tok)
		assert.Equal(t, err, expected, "%d", status)
		assert.Assert(t, newToken == nil)
	}
}

func ptrTo[T any](v T) *T {
	return &v
}

func TestUpdateChatSettings(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const broadcasterID = 1
	const modID = 123
	tok := tokFor(ctx, t, tw, ft, modID)

	newToken, err := tw.UpdateChatSettings(ctx, broadcasterID, modID, tok, &twitch.ChatSettingsPatch{
		EmoteMode: ptrTo(true),
	})

	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)
}

func TestUpdateChatSettingsBadParameters(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const broadcasterID = 1
	const modID = 123
	tok := tokFor(ctx, t, tw, ft, modID)

	_, err := tw.UpdateChatSettings(ctx, broadcasterID, modID, tok, nil)

	assert.ErrorIs(t, err, twitch.ErrBadRequest)

	_, err = tw.UpdateChatSettings(ctx, broadcasterID, modID, tok, &twitch.ChatSettingsPatch{})

	assert.ErrorIs(t, err, twitch.ErrBadRequest)

	_, err = tw.UpdateChatSettings(ctx, broadcasterID, modID, tok, &twitch.ChatSettingsPatch{
		FollowerModeDuration: ptrTo(int64(30)),
	})

	assert.ErrorIs(t, err, twitch.ErrBadRequest)

	_, err = tw.UpdateChatSettings(ctx, broadcasterID, modID, nil, &twitch.ChatSettingsPatch{
		EmoteMode: ptrTo(true),
	})

	assert.ErrorIs(t, err, twitch.ErrNotAuthorized)

	_, err = tw.UpdateChatSettings(ctx, broadcasterID, modID, &oauth2.Token{}, &twitch.ChatSettingsPatch{
		EmoteMode: ptrTo(true),
	})

	assert.ErrorIs(t, err, twitch.ErrNotAuthorized)
}

func TestUpdateChatSettingsErrors(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const broadcasterID = 1
	const modID = 123
	tok := tokFor(ctx, t, tw, ft, modID)

	banRequest := &twitch.ChatSettingsPatch{
		EmoteMode: ptrTo(true),
	}

	_, err := tw.UpdateChatSettings(ctx, 777, modID, tok, banRequest)
	assert.ErrorContains(t, err, errTestBadRequest.Error())

	for status, expected := range expectedErrors {
		id := int64(status)
		tok := tokFor(ctx, t, tw, ft, id)

		newToken, err := tw.UpdateChatSettings(ctx, id, modID, tok, banRequest)
		assert.Equal(t, err, expected, "%d", status)
		assert.Assert(t, newToken == nil)
	}
}

func TestAnnounce(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const broadcasterID = 1
	const modID = 123
	tok := tokFor(ctx, t, tw, ft, modID)

	newToken, err := tw.Announce(ctx, broadcasterID, modID, tok, "Some announcement!", "purple")

	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)
}

func TestAnnounceBadParameters(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const broadcasterID = 1
	const modID = 123
	tok := tokFor(ctx, t, tw, ft, modID)

	_, err := tw.Announce(ctx, broadcasterID, modID, tok, "", "purple")
	assert.ErrorIs(t, err, twitch.ErrBadRequest)

	_, err = tw.Announce(ctx, broadcasterID, modID, nil, "Some announcement!", "purple")
	assert.ErrorIs(t, err, twitch.ErrNotAuthorized)

	_, err = tw.Announce(ctx, broadcasterID, modID, &oauth2.Token{}, "Some announcement!", "purple")
	assert.ErrorIs(t, err, twitch.ErrNotAuthorized)
}

func TestAnnounceErrors(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const broadcasterID = 1
	const modID = 123
	tok := tokFor(ctx, t, tw, ft, modID)

	_, err := tw.Announce(ctx, 777, modID, tok, "Some announcement!", "purple")
	assert.ErrorContains(t, err, errTestBadRequest.Error())

	for status, expected := range expectedErrors {
		id := int64(status)
		tok := tokFor(ctx, t, tw, ft, id)

		newToken, err := tw.Announce(ctx, id, modID, tok, "Some announcement!", "purple")
		assert.Equal(t, err, expected, "%d", status)
		assert.Assert(t, newToken == nil)
	}
}
