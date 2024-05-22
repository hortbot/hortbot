package twitch

import (
	"context"
	"net/url"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
)

type Conduit struct {
	ID         string `json:"id"`
	ShardCount int    `json:"shard_count"`
}

const helixEventsubConduit = helixRoot + "/eventsub/conduits"

func (t *Twitch) GetConduits(ctx context.Context) ([]*Conduit, error) {
	cli := t.helixCli
	return fetchList[*Conduit](ctx, cli, helixEventsubConduit)
}

func (t *Twitch) CreateConduit(ctx context.Context, shardCount int) (*Conduit, error) {
	cli := t.helixCli
	body := &struct {
		ShardCount int `json:"shard_count"`
	}{
		ShardCount: shardCount,
	}

	conduit, err := postAndDecodeFirstFromList[*Conduit](ctx, cli, helixEventsubConduit, body)
	if err != nil {
		return nil, err
	}
	return conduit, nil
}

func (t *Twitch) UpdateConduit(ctx context.Context, id string, shardCount int) error {
	cli := t.helixCli
	body := &struct {
		ID         string `json:"id"`
		ShardCount int    `json:"shard_count"`
	}{
		ID:         id,
		ShardCount: shardCount,
	}

	resp, err := cli.Patch(ctx, helixEventsubConduit, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return statusToError(resp.StatusCode)
}

func (t *Twitch) DeleteConduit(ctx context.Context, id string) error {
	cli := t.helixCli
	url := helixEventsubConduit + "?id=" + id
	resp, err := cli.Delete(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return statusToError(resp.StatusCode)
}

type Shard struct {
	ID        idstr.IDStr        `json:"id"`
	Transport eventsub.Transport `json:"transport"`
}

const helixEventsubShards = helixRoot + "/eventsub/conduits/shards"

func (t *Twitch) UpdateShards(ctx context.Context, conduitID string, shards []*Shard) error {
	cli := t.helixCli
	body := &struct {
		ConduitID string   `json:"conduit_id"`
		Shards    []*Shard `json:"shards"`
	}{
		ConduitID: conduitID,
		Shards:    shards,
	}

	resp, err := cli.Patch(ctx, helixEventsubShards, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return statusToError(resp.StatusCode)
}

const helixEventsubSubscriptions = helixRoot + "/eventsub/subscriptions"

func (t *Twitch) GetSubscriptions(ctx context.Context) ([]*eventsub.Subscription, error) {
	cli := t.helixCli
	return paginate[*eventsub.Subscription](ctx, cli, helixEventsubSubscriptions, url.Values{}, 0, 10000)
}

func (t *Twitch) DeleteSubscription(ctx context.Context, id string) error {
	cli := t.helixCli
	url := helixEventsubSubscriptions + "?id=" + id
	resp, err := cli.Delete(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return statusToError(resp.StatusCode)
}

func (t *Twitch) CreateChatSubscription(ctx context.Context, conduitID string, broadcasterID int64, botID int64) error {
	body := struct {
		Type      string                                    `json:"type"`
		Version   string                                    `json:"version"`
		Condition eventsub.ChatMessageSubscriptionCondition `json:"condition"`
		Transport eventsub.Transport                        `json:"transport"`
	}{
		Type:    eventsub.ChatMessageSubscriptionType,
		Version: "1",
		Condition: eventsub.ChatMessageSubscriptionCondition{
			BroadcasterUserID: idstr.IDStr(broadcasterID),
			UserID:            idstr.IDStr(botID),
		},
		Transport: eventsub.Transport{
			Method:    "conduit",
			ConduitID: conduitID,
		},
	}

	cli := t.helixCli
	resp, err := cli.Post(ctx, helixEventsubSubscriptions, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return statusToError(resp.StatusCode)
}
