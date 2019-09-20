package twitch

import (
	"context"
	"strconv"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2"
)

type CachedAPI struct {
	api   API
	cache *cache.Cache
}

var _ API = (*CachedAPI)(nil)

func NewCached(api API) *CachedAPI {
	return &CachedAPI{
		api:   api,
		cache: cache.New(time.Minute, 5*time.Minute),
	}
}

func (ca *CachedAPI) Flush() {
	ca.cache.Flush()
}

func (ca *CachedAPI) GetUserForUsername(ctx context.Context, username string) (*User, error) {
	key := "GetUserForUsername/" + username

	type result struct {
		u   *User
		err error
	}

	cached, ok := ca.cache.Get(key)
	if ok {
		r := cached.(result)
		return r.u, r.err
	}

	user, err := ca.api.GetUserForUsername(ctx, username)
	ca.cache.SetDefault(key, result{u: user, err: err})
	return user, err
}

func (ca *CachedAPI) GetChannelByID(ctx context.Context, id int64) (*Channel, error) {
	key := "GetChannelByID/" + strconv.FormatInt(id, 10)

	type result struct {
		c   *Channel
		err error
	}

	cached, ok := ca.cache.Get(key)
	if ok {
		r := cached.(result)
		return r.c, r.err
	}

	c, err := ca.api.GetChannelByID(ctx, id)
	ca.cache.SetDefault(key, result{c: c, err: err})
	return c, err
}

func (ca *CachedAPI) GetCurrentStream(ctx context.Context, id int64) (*Stream, error) {
	key := "GetCurrentStream/" + strconv.FormatInt(id, 10)

	type result struct {
		s   *Stream
		err error
	}

	cached, ok := ca.cache.Get(key)
	if ok {
		r := cached.(result)
		return r.s, r.err
	}

	s, err := ca.api.GetCurrentStream(ctx, id)
	ca.cache.SetDefault(key, result{s: s, err: err})
	return s, err
}

func (ca *CachedAPI) GetChatters(ctx context.Context, channel string) (*Chatters, error) {
	return ca.api.GetChatters(ctx, channel)
}

func (ca *CachedAPI) GetUserForToken(ctx context.Context, userToken *oauth2.Token) (user *User, newToken *oauth2.Token, err error) {
	return ca.api.GetUserForToken(ctx, userToken)
}

func (ca *CachedAPI) SetChannelStatus(ctx context.Context, id int64, userToken *oauth2.Token, status string) (string, *oauth2.Token, error) {
	ca.cache.Delete("GetChannelByID/" + strconv.FormatInt(id, 10))
	ca.cache.Delete("GetCurrentStream/" + strconv.FormatInt(id, 10))
	return ca.api.SetChannelStatus(ctx, id, userToken, status)
}

func (ca *CachedAPI) SetChannelGame(ctx context.Context, id int64, userToken *oauth2.Token, game string) (string, *oauth2.Token, error) {
	ca.cache.Delete("GetChannelByID/" + strconv.FormatInt(id, 10))
	ca.cache.Delete("GetCurrentStream/" + strconv.FormatInt(id, 10))
	return ca.api.SetChannelGame(ctx, id, userToken, game)
}

func (ca *CachedAPI) FollowChannel(ctx context.Context, id int64, userToken *oauth2.Token, toFollow int64) (newToken *oauth2.Token, err error) {
	return ca.api.FollowChannel(ctx, id, userToken, toFollow)
}
