package conduit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

const initialWebsocketURL = "wss://eventsub.wss.twitch.tv/ws"

// NotificationHandler processes an EventSub notification before the websocket
// reader accepts another message.
type NotificationHandler func(context.Context, json.RawMessage, *eventsub.WebsocketMessage) error

type Service struct {
	db                 *pgxpool.Pool
	queries            *dbsql.Queries
	twitch             twitch.API
	syncInterval       time.Duration
	shards             int
	handleNotification NotificationHandler

	g *errgroupx.Group

	started     chan struct{}
	startedOnce sync.Once

	conduitID      string
	websocketCount atomic.Int64
	shardMu        sync.Mutex
}

type chatSubscription struct {
	BroadcasterID int64
	BotID         int64
}

func New(
	db *pgxpool.Pool,
	twitch twitch.API,
	syncInterval time.Duration,
	shards int,
	handleNotification NotificationHandler,
) *Service {
	if handleNotification == nil {
		panic("nil notification handler")
	}
	return &Service{
		db:                 db,
		queries:            dbsql.New(db),
		twitch:             twitch,
		syncInterval:       syncInterval,
		shards:             shards,
		handleNotification: handleNotification,
		started:            make(chan struct{}),
	}
}

func (s *Service) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	conduit, err := s.getOrCreateConduit(ctx)
	if err != nil {
		return err
	}
	s.conduitID = conduit.ID

	ctx = ctxlog.With(ctx, zap.String("conduit_id", s.conduitID))
	ctxlog.Info(ctx, "spinning up shards")

	s.g = errgroupx.FromContext(ctx)

	for i := range s.shards {
		s.g.Go(func(ctx context.Context) error {
			return s.runWebsocket(ctx, initialWebsocketURL, i, s.onStart)
		})
	}

	return s.g.WaitIgnoreStop()
}

func (s *Service) onStart() {
	s.startedOnce.Do(func() { close(s.started) })
}

func (s *Service) getOrCreateConduit(ctx context.Context) (*twitch.Conduit, error) {
	conduits, err := s.twitch.GetConduits(ctx)
	if err != nil {
		if ae, ok := apiclient.AsError(err); !ok || !ae.IsNotFound() {
			return nil, fmt.Errorf("get conduits: %w", err)
		}
	}

	if len(conduits) == 0 {
		ctxlog.Info(ctx, "creating conduit")
		conduit, err := s.twitch.CreateConduit(ctx, s.shards)
		if err != nil {
			return nil, fmt.Errorf("create conduit: %w", err)
		}
		return conduit, nil
	}

	conduit := conduits[0]
	if conduit.ShardCount != s.shards {
		ctxlog.Info(ctx, "reusing conduit but updating shard count", zap.Int("shardCount", s.shards))
		conduit, err := s.twitch.UpdateConduit(ctx, conduit.ID, s.shards)
		if err != nil {
			return nil, fmt.Errorf("update conduit: %w", err)
		}
		return conduit, nil
	}

	ctxlog.Info(ctx, "reusing conduit")
	return conduit, nil
}

func (s *Service) setConduitShardSession(ctx context.Context, shard int, sessionID string) error {
	s.shardMu.Lock()
	defer s.shardMu.Unlock()

	ctxlog.Info(ctx, "setting conduit shard session", zap.String("sessionID", sessionID))
	if err := s.twitch.UpdateShards(ctx, s.conduitID, []*twitch.Shard{
		{
			ID: idstr.IDStr(shard),
			Transport: eventsub.Transport{
				Method:    "websocket",
				SessionID: sessionID,
			},
		},
	}); err != nil {
		return fmt.Errorf("update shards: %w", err)
	}
	return nil
}

func (s *Service) runWebsocket(ctx context.Context, url string, shard int, onWelcome func()) error {
	for ctx.Err() == nil {
		if err := s.runOneWebsocket(ctx, url, shard, onWelcome); err != nil {
			if errors.Is(err, errWebsocketClosedForReconnect) {
				ctxlog.Info(ctx, "websocket closed for reconnect")
				return nil
			}
			ctxlog.Warn(ctx, "websocket error, restarting", zap.Error(err))
			// Reset the URL; the reconnect URL is not going to work.
			url = initialWebsocketURL
		}
		onWelcome = nil
		metricDisconnects.Inc()

		if ctx.Err() != nil {
			return ctx.Err()
		}

		const wait = time.Second
		ctxlog.Info(ctx, "waiting before reconnect", zap.Duration("wait", wait))

		select {
		case <-time.After(wait):
		case <-ctx.Done():
		}
	}

	return ctx.Err()
}

var errWebsocketClosedForReconnect = errors.New("websocket closed")

func (s *Service) runOneWebsocket(ctx context.Context, url string, shard int, onWelcome func()) error {
	ctx = ctxlog.With(ctx, zap.Int("shard_id", shard))

	ctxlog.Info(ctx, "creating websocket")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	metricWebsockets.Set(float64(s.websocketCount.Add(1)))
	defer func() { metricWebsockets.Set(float64(s.websocketCount.Add(-1))) }()

	c, response, err := websocket.Dial(ctx, url, nil)
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}
	defer c.CloseNow() //nolint:errcheck

	reconnecting := false
	readTimeout := 30 * time.Second

	for ctx.Err() == nil {
		beforeRead := time.Now()
		var raw json.RawMessage
		readCtx, cancelRead := context.WithTimeout(ctx, readTimeout)
		err := wsjson.Read(readCtx, c, &raw)
		cancelRead()
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			if reconnecting {
				return errWebsocketClosedForReconnect
			}
			if errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("eventsub keepalive timed out after %s: %w", readTimeout, err)
			}
			return fmt.Errorf("read websocket: %w", err)
		}
		metricWebsocketReadDuration.Observe(time.Since(beforeRead).Seconds())

		var msg eventsub.WebsocketMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			field := "unknown"
			value := "unknown"
			if ue, ok := errors.AsType[*eventsub.UnknownTypeError](err); ok {
				field = ue.Field
				value = ue.Value
			}
			metricDecodeErrors.WithLabelValues(field, value).Inc()
			ctxlog.Warn(ctx, "websocket unmarshal error", zap.Error(err), zap.ByteString("raw", raw))
			continue
		}

		metricHandled.WithLabelValues(msg.Metadata.MessageType).Inc()

		switch payload := msg.Payload.(type) {
		case *eventsub.SessionWelcomePayload:
			if payload.Session.KeepaliveTimeoutSeconds > 0 {
				readTimeout = time.Duration(payload.Session.KeepaliveTimeoutSeconds)*time.Second + time.Second
			}
			if err := s.setConduitShardSession(ctx, shard, payload.Session.ID); err != nil {
				return err
			}
			if onWelcome != nil {
				onWelcome()
				onWelcome = nil
			}
		case *eventsub.SessionReconnectPayload:
			metricReconnects.Inc()
			reconnecting = true
			s.g.Go(func(ctx context.Context) error {
				return s.runWebsocket(ctx, *payload.Session.ReconnectURL, shard, cancel)
			})
		case *eventsub.NotificationPayload:
			if err := s.handleNotification(ctx, raw, &msg); err != nil {
				return fmt.Errorf("handle notification: %w", err)
			}
		}
	}

	if err := c.Close(websocket.StatusNormalClosure, ""); err != nil {
		ctxlog.Debug(ctx, "websocket close error", zap.Error(err))
	}

	if reconnecting {
		return errWebsocketClosedForReconnect
	}

	return nil
}

var possibleStatuses = []string{
	"enabled",
	"webhook_callback_verification_pending",
	"webhook_callback_verification_failed",
	"notification_failures_exceeded",
	"authorization_revoked",
	"moderator_removed",
	"user_removed",
	"chat_user_banned",
	"version_removed",
	"beta_maintenance",
	"websocket_disconnected",
	"websocket_failed_ping_pong",
	"websocket_received_inbound_traffic",
	"websocket_connection_unused",
	"websocket_internal_error",
	"websocket_network_timeout",
	"websocket_network_error",
	"websocket_failed_to_reconnect",
}

func (s *Service) SynchronizeSubscriptions(ctx context.Context) error {
	start := time.Now()
	defer func() { metricSyncDuration.Observe(time.Since(start).Seconds()) }()

	select {
	case <-s.started:
	case <-ctx.Done():
		return ctx.Err()
	}

	ctxlog.Debug(ctx, "synchronizing subscriptions")

	channels, err := s.queries.ActiveChannelsByBot(ctx)
	if err != nil {
		return fmt.Errorf("list active eventsub channels: %w", err)
	}

	wanted := make(map[chatSubscription]struct{})
	for botID, broadcasterIDs := range channels {
		for _, broadcasterID := range broadcasterIDs {
			wanted[chatSubscription{
				BroadcasterID: broadcasterID,
				BotID:         botID,
			}] = struct{}{}
		}
	}
	metricWantedChatSubscriptions.Set(float64(len(wanted)))

	allSubscriptions, err := s.twitch.GetSubscriptions(ctx)
	if err != nil {
		if ae, ok := apiclient.AsError(err); !ok || !ae.IsNotFound() {
			return fmt.Errorf("get subscriptions: %w", err)
		}
	}

	if len(allSubscriptions) == 0 {
		ctxlog.Warn(ctx, "no subscriptions found")
	}

	metricSubscriptions.Set(float64(len(allSubscriptions)))

	actual, stale, statuses := classifyChatSubscriptions(ctx, s.conduitID, allSubscriptions)
	metricCurrentChatSubscriptions.Set(float64(len(actual)))

	for _, status := range possibleStatuses {
		metricSubscriptionTypes.WithLabelValues(status).Set(float64(statuses[status]))
	}

	for sub := range actual {
		if _, ok := wanted[sub]; ok {
			delete(wanted, sub)
			delete(actual, sub)
		}
	}

	// wanted now contains subscriptions to create, actual contains enabled
	// subscriptions to remove, and stale contains disabled or duplicate ones.
	toCreate := wanted
	toDelete := actual
	metricCreateChatSubscriptions.Set(float64(len(toCreate)))
	metricDeleteChatSubscriptions.Set(float64(len(stale) + len(toDelete)))

	ctxlog.Debug(ctx, "synchronizing subscriptions",
		zap.Int("subscriptions", len(allSubscriptions)),
		zap.Int("add_count", len(toCreate)),
		zap.Int("remove_count", len(stale)+len(toDelete)),
		zap.Any("add", keys(toCreate)),
		zap.Any("remove_stale", stale),
		zap.Any("remove", keys(toDelete)),
	)

	for id, sub := range stale {
		if err := s.twitch.DeleteSubscription(ctx, id); err != nil {
			ctxlog.Warn(ctx, "delete subscription error", zap.Error(err), zap.Any("subscription", sub), zap.String("id", id))
			metricDeleteSubscriptionErrors.Inc()
		} else {
			metricDeletedSubscriptions.Inc()
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	for sub := range toCreate {
		if sub.BotID == 0 {
			ctxlog.Error(ctx, "subscription has no bot ID", zap.Any("subscription", sub))
			continue
		}

		if err := s.twitch.CreateChatSubscription(ctx, s.conduitID, sub.BroadcasterID, sub.BotID); err != nil {
			ctxlog.Warn(ctx, "create subscription error", zap.Error(err), zap.Any("subscription", sub))
			metricCreateSubscriptionErrors.Inc()
		} else {
			metricCreatedSubscriptions.Inc()
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	for sub, id := range toDelete {
		if err := s.twitch.DeleteSubscription(ctx, id); err != nil {
			ctxlog.Warn(ctx, "delete subscription error", zap.Error(err), zap.Any("subscription", sub), zap.String("id", id))
			metricDeleteSubscriptionErrors.Inc()
		} else {
			metricDeletedSubscriptions.Inc()
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return nil
}

func classifyChatSubscriptions(ctx context.Context, conduitID string, subscriptions []*eventsub.Subscription) (actual map[chatSubscription]string, stale map[string]chatSubscription, statuses map[string]int) {
	actual = make(map[chatSubscription]string, len(subscriptions))
	stale = make(map[string]chatSubscription)
	statuses = make(map[string]int, len(subscriptions))

	for _, sub := range subscriptions {
		statuses[sub.Status]++

		if sub.Transport.ConduitID != conduitID {
			ctxlog.Warn(ctx, "subscription not using our conduit",
				zap.String("id", sub.ID),
				zap.Any("transport", sub.Transport),
			)
			continue
		}
		if sub.Type != eventsub.ChatMessageSubscriptionType {
			continue
		}

		condition := sub.Condition.(*eventsub.ChatMessageSubscriptionCondition)
		chatSub := chatSubscription{
			BroadcasterID: int64(condition.BroadcasterUserID),
			BotID:         int64(condition.UserID),
		}
		if sub.Status != "enabled" {
			stale[sub.ID] = chatSub
			continue
		}
		if _, ok := actual[chatSub]; ok {
			stale[sub.ID] = chatSub
			continue
		}
		actual[chatSub] = sub.ID
	}

	return actual, stale, statuses
}

func keys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}
