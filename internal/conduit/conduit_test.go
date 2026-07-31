package conduit

import (
	"context"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
	"gotest.tools/v3/assert"
)

func TestDisabledSubscriptionIsNotCurrent(t *testing.T) {
	t.Parallel()

	sub := &eventsub.Subscription{
		ID:     "revoked",
		Status: "authorization_revoked",
		Type:   eventsub.ChatMessageSubscriptionType,
		Condition: &eventsub.ChatMessageSubscriptionCondition{
			BroadcasterUserID: idstr.IDStr(1),
			UserID:            idstr.IDStr(2),
		},
		Transport: &eventsub.Transport{
			ConduitID: "conduit",
		},
	}

	chatSub := chatSubscription{BroadcasterID: 1, BotID: 2}
	actual, stale, statuses := classifyChatSubscriptions(context.Background(), "conduit", []*eventsub.Subscription{sub})

	assert.DeepEqual(t, actual, map[chatSubscription]string{})
	assert.DeepEqual(t, stale, map[string]chatSubscription{"revoked": chatSub})
	assert.DeepEqual(t, statuses, map[string]int{"authorization_revoked": 1})
}
