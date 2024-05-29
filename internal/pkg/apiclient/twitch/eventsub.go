package twitch

import (
	"context"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
)

type Conduit struct {
	ID         string `json:"id"`
	ShardCount int    `json:"shard_count"`
}

const helixEventsubConduit = helixRoot + "/eventsub/conduits"

func (t *Twitch) GetConduits(ctx context.Context) ([]*Conduit, error) {
	req, err := t.helixCli.NewRequest(ctx, helixEventsubConduit)
	if err != nil {
		return nil, err
	}
	return fetchList[*Conduit](ctx, req)
}

func (t *Twitch) CreateConduit(ctx context.Context, shardCount int) (*Conduit, error) {
	body := &struct {
		ShardCount int `json:"shard_count"`
	}{
		ShardCount: shardCount,
	}

	req, err := t.helixCli.NewRequest(ctx, helixEventsubConduit)
	if err != nil {
		return nil, err
	}
	req.BodyJSON(body).Post()

	return fetchFirstFromList[*Conduit](ctx, req)
}

func (t *Twitch) UpdateConduit(ctx context.Context, id string, shardCount int) (*Conduit, error) {
	body := &struct {
		ID         string `json:"id"`
		ShardCount int    `json:"shard_count"`
	}{
		ID:         id,
		ShardCount: shardCount,
	}

	req, err := t.helixCli.NewRequest(ctx, helixEventsubConduit)
	if err != nil {
		return nil, err
	}
	req.BodyJSON(body).Patch()

	return fetchFirstFromList[*Conduit](ctx, req)
}

func (t *Twitch) DeleteConduit(ctx context.Context, id string) error {
	req, err := t.helixCli.NewRequest(ctx, helixEventsubConduit)
	if err != nil {
		return err
	}
	if err := req.Param("id", id).Delete().Fetch(ctx); err != nil {
		return apiclient.WrapRequestErr("twitch", err, nil)
	}
	return nil
}

type Shard struct {
	ID        idstr.IDStr        `json:"id"`
	Transport eventsub.Transport `json:"transport"`
}

const helixEventsubShards = helixRoot + "/eventsub/conduits/shards"

func (t *Twitch) UpdateShards(ctx context.Context, conduitID string, shards []*Shard) error {
	body := &struct {
		ConduitID string   `json:"conduit_id"`
		Shards    []*Shard `json:"shards"`
	}{
		ConduitID: conduitID,
		Shards:    shards,
	}

	req, err := t.helixCli.NewRequest(ctx, helixEventsubShards)
	if err != nil {
		return err
	}

	if err := req.BodyJSON(body).Patch().Fetch(ctx); err != nil {
		return apiclient.WrapRequestErr("twitch", err, nil)
	}
	return nil
}

const helixEventsubSubscriptions = helixRoot + "/eventsub/subscriptions"

func (t *Twitch) GetSubscriptions(ctx context.Context) ([]*eventsub.Subscription, error) {
	req, err := t.helixCli.NewRequest(ctx, helixEventsubSubscriptions)
	if err != nil {
		return nil, err
	}
	return paginate[*eventsub.Subscription](ctx, req, 0, 10000)
}

func (t *Twitch) DeleteSubscription(ctx context.Context, id string) error {
	req, err := t.helixCli.NewRequest(ctx, helixEventsubSubscriptions)
	if err != nil {
		return err
	}
	if err := req.Param("id", id).Delete().Fetch(ctx); err != nil {
		return apiclient.WrapRequestErr("twitch", err, nil)
	}
	return nil
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

	req, err := t.helixCli.NewRequest(ctx, helixEventsubSubscriptions)
	if err != nil {
		return err
	}
	if err := req.BodyJSON(body).Post().Fetch(ctx); err != nil {
		return apiclient.WrapRequestErr("twitch", err, nil)
	}
	return nil
}
