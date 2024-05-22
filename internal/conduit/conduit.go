package conduit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/errorsx"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Service struct {
	db           *sql.DB
	twitch       twitch.API
	syncInterval time.Duration

	g *errgroupx.Group

	started  chan struct{}
	incoming chan *eventsub.WebsocketMessage

	conduitID      string
	websocketCount atomic.Int64
}

func New(db *sql.DB, twitch twitch.API, syncInterval time.Duration) *Service {
	return &Service{
		db:           db,
		twitch:       twitch,
		syncInterval: syncInterval,
		started:      make(chan struct{}),
		incoming:     make(chan *eventsub.WebsocketMessage, 10),
	}
}

func (s *Service) Incoming() <-chan *eventsub.WebsocketMessage {
	return s.incoming
}

func (s *Service) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	s.g = errgroupx.FromContext(ctx)

	conduit, err := s.getOrCreateConduit(ctx)
	if err != nil {
		return err
	}
	s.conduitID = conduit.ID

	ctxlog.Info(ctx, "conduit created", zap.String("id", s.conduitID))

	s.g.Go(s.runWebsocket)

	return s.g.WaitIgnoreStop()
}

func (s *Service) getOrCreateConduit(ctx context.Context) (*twitch.Conduit, error) {
	conduits, err := s.twitch.GetConduits(ctx)
	if err != nil && !errors.Is(err, twitch.ErrNotFound) {
		return nil, fmt.Errorf("get conduits: %w", err)
	}

	if len(conduits) == 0 {
		ctxlog.Info(ctx, "creating conduit")
		conduit, err := s.twitch.CreateConduit(ctx, 1)
		if err != nil {
			return nil, fmt.Errorf("create conduit: %w", err)
		}
		return conduit, nil
	}

	return conduits[0], nil
}

func (s *Service) setConduitShardSession(ctx context.Context, sessionID string) error {
	ctxlog.Info(ctx, "setting conduit shard session", zap.String("sessionID", sessionID))
	if err := s.twitch.UpdateShards(ctx, s.conduitID, []*twitch.Shard{
		{
			ID: idstr.IDStr(0),
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

func (s *Service) runWebsocket(ctx context.Context) error {
	onWelcome := func() { close(s.started) }

	for ctx.Err() == nil {
		if err := s.runOneWebsocket(ctx, "wss://eventsub.wss.twitch.tv/ws", onWelcome); err != nil {
			ctxlog.Error(ctx, "websocket error, restarting", zap.Error(err))
		}
		onWelcome = nil
		metricDisconnects.Inc()

		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return ctx.Err()
}

func (s *Service) runOneWebsocket(ctx context.Context, url string, onWelcome func()) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	metricWebsockets.Set(float64(s.websocketCount.Add(1)))
	defer func() { metricWebsockets.Set(float64(s.websocketCount.Add(-1))) }()

	c, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}
	defer c.CloseNow() //nolint:errcheck

readLoop:
	for ctx.Err() == nil {
		var raw json.RawMessage
		if err := wsjson.Read(ctx, c, &raw); err != nil {
			ctxlog.Warn(ctx, "websocket read error", zap.Error(err))
			break readLoop
		}

		var msg eventsub.WebsocketMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			field := "unknown"
			value := "unknown"
			if ue, ok := errorsx.As[*eventsub.UnknownTypeError](err); ok {
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
			if err := s.setConduitShardSession(ctx, payload.Session.ID); err != nil {
				return err
			}
			if onWelcome != nil {
				onWelcome()
				onWelcome = nil
			}
		case *eventsub.SessionReconnectPayload:
			s.g.Go(func(ctx context.Context) error {
				return s.runOneWebsocket(ctx, *payload.Session.ReconnectURL, cancel)
			})
		case *eventsub.NotificationPayload:
			select {
			case s.incoming <- &msg:
			case <-ctx.Done():
				break readLoop
			}
		}
	}

	if err := c.Close(websocket.StatusNormalClosure, ""); err != nil {
		ctxlog.Debug(ctx, "websocket close error", zap.Error(err))
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

	channels, err := modelsx.ListActiveEventSubChannels(ctx, s.db)
	if err != nil {
		return fmt.Errorf("list active eventsub channels: %w", err)
	}

	// exported fields for logging
	type subscription struct {
		BroadcasterID int64
		BotID         int64
	}

	wanted := make(map[subscription]struct{})
	for botID, broadcasterIDs := range channels {
		for _, broadcasterID := range broadcasterIDs {
			wanted[subscription{
				BroadcasterID: broadcasterID,
				BotID:         botID,
			}] = struct{}{}
		}
	}

	allSubscriptions, err := s.twitch.GetSubscriptions(ctx)
	if err != nil && !errors.Is(err, twitch.ErrNotFound) {
		return fmt.Errorf("get subscriptions: %w", err)
	}

	metricSubscriptions.Set(float64(len(allSubscriptions)))

	statuses := make(map[string]int, len(allSubscriptions))

	actual := make(map[subscription]string, len(allSubscriptions))
	for _, sub := range allSubscriptions {
		statuses[sub.Status]++

		if sub.Transport.ConduitID != s.conduitID {
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
		actual[subscription{
			BroadcasterID: int64(condition.BroadcasterUserID),
			BotID:         int64(condition.UserID),
		}] = sub.ID
	}

	for _, status := range possibleStatuses {
		metricSubscriptionTypes.WithLabelValues(status).Set(float64(statuses[status]))
	}

	for sub := range wanted {
		if _, ok := actual[sub]; ok {
			delete(actual, sub)
			delete(wanted, sub)
		}
	}

	for sub := range actual {
		if _, ok := wanted[sub]; ok {
			delete(actual, sub)
			delete(wanted, sub)
		}
	}

	// wanted now contains toCreate subscriptions, actual contains extra subscriptions
	toCreate := wanted
	toDelete := actual

	ctxlog.Info(ctx, "synchronizing subscriptions",
		zap.Int("subscriptions", len(allSubscriptions)),
		zap.Int("add_count", len(toCreate)),
		zap.Int("remove_count", len(toDelete)),
		zap.Any("add", keys(toCreate)),
		zap.Any("remove", keys(toDelete)),
	)

	metricCreatedSubscriptions.Add(float64(len(toCreate)))
	metricDeletedSubscriptions.Add(float64(len(toDelete)))

	for sub := range toCreate {
		if sub.BotID == 0 {
			ctxlog.Error(ctx, "subscription has no bot ID", zap.Any("subscription", sub))
			continue
		}

		if err := s.twitch.CreateChatSubscription(ctx, s.conduitID, sub.BroadcasterID, sub.BotID); err != nil {
			ctxlog.Warn(ctx, "create subscription error", zap.Error(err), zap.Any("subscription", sub))
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	for sub, id := range toDelete {
		if err := s.twitch.DeleteSubscription(ctx, id); err != nil {
			ctxlog.Warn(ctx, "delete subscription error", zap.Error(err), zap.Any("subscription", sub), zap.String("id", id))
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return nil
}

func keys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}
