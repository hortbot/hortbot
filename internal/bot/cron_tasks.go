package bot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/dbx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

func (b *Bot) validateTokens(ctx context.Context, log bool) error {
	logFn := ctxlog.Debug
	if log {
		logFn = ctxlog.Info
	}

	logFn(ctx, "validating twitch tokens")
	start := time.Now()

	tokens, err := b.queries.ListTwitchTokens(ctx)
	if err != nil {
		return fmt.Errorf("getting tokens: %w", err)
	}

	validated := 0
	updated := 0
	deleted := 0

	for _, tt := range tokens {
		err := dbx.Transact(ctx, b.db, func(ctx context.Context, tx pgx.Tx) error {
			q := dbsql.New(tx)
			ctx = ctxlog.With(ctx, zap.Int64("twitch_id", tt.TwitchID))

			ctxlog.Debug(ctx, "validating token")
			token := tt.OAuth2Token()
			validation, newToken, err := b.deps.Twitch.Validate(ctx, token)
			if err != nil {
				if te, ok := apiclient.AsError(err); ok {
					if errors.Is(err, twitch.ErrDeadToken) || te.IsNotPermitted() || te.IsNotFound() {
						ctxlog.Info(ctx, "deleting dead token", zap.Error(err), zap.Int64("twitch_id", tt.TwitchID), zap.String("bot_name", tt.BotName.String))
						if err := q.DeleteTwitchTokenByID(ctx, tt.TwitchID); err != nil {
							return fmt.Errorf("deleting dead token: %w", err)
						}
						metricDeletedTokens.Inc()
						deleted++
						return nil
					}
				}

				ctxlog.Error(ctx, "failed to validate token", zap.Error(err))
				metricTokenValidationErrors.Inc()
				return nil
			}

			if newToken != nil {
				tt = *dbsql.NewTwitchToken(newToken, tt.TwitchID, pgtype.Text{
					String: tt.BotName.String,
					Valid:  tt.BotName.Valid,
				}, validation.Scopes)
			} else {
				tt.Scopes = validation.Scopes
			}

			ctxlog.Debug(ctx, "token validated", zap.Bool("new_token", newToken != nil), zap.Strings("scopes", tt.Scopes))
			if err := q.SaveTwitchToken(ctx, &tt); err != nil {
				return fmt.Errorf("upserting token: %w", err)
			}

			validated++
			metricValidatedTokens.Inc()
			if newToken != nil {
				updated++
				metricUpdatedTokens.Inc()
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	logFn(ctx, "validated twitch tokens",
		zap.Duration("duration", time.Since(start)),
		zap.Int("total", len(tokens)),
		zap.Int("validated", validated),
		zap.Int("updated", updated),
		zap.Int("deleted", deleted),
	)
	return nil
}

func (b *Bot) runValidateTokens(ctx context.Context) error {
	for {
		log := false
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tickerChan(b.validateTokensTicker):
		case <-b.validateTokensManual:
			log = true
		}

		if err := b.validateTokens(ctx, log); err != nil {
			ctxlog.Error(ctx, "failed to validate tokens", zap.Error(err))
		}
	}
}

func (b *Bot) triggerValidateTokensNow() {
	select {
	case b.validateTokensManual <- struct{}{}:
	default:
	}
}

func (b *Bot) updateModeratedChannels(ctx context.Context, log bool) error {
	logFn := ctxlog.Debug
	if log {
		logFn = ctxlog.Info
	}

	logFn(ctx, "updating moderated channels")
	start := time.Now()

	err := dbx.Transact(ctx, b.db, func(ctx context.Context, tx pgx.Tx) error {
		q := dbsql.New(tx)
		botTokens, err := q.ListModerationBotTwitchTokens(ctx)
		if err != nil {
			return fmt.Errorf("getting bot tokens: %w", err)
		}

		logFn(ctx, "locking moderated_channels table")
		if err := q.LockModeratedChannels(ctx); err != nil {
			return fmt.Errorf("locking moderated_channels: %w", err)
		}

		start := time.Now()

		for _, botToken := range botTokens {
			botName := botToken.BotName.String

			logFn(ctx, "updating bot", zap.String("bot_name", botName))
			token := botToken.OAuth2Token()
			moderatedChannels, newToken, err := b.deps.Twitch.GetModeratedChannels(ctx, botToken.TwitchID, token)
			if newToken != nil {
				botToken := dbsql.NewTwitchToken(newToken, botToken.TwitchID, pgtype.Text{
					String: botToken.BotName.String,
					Valid:  botToken.BotName.Valid,
				}, botToken.Scopes)
				if err := dbsql.New(tx).SaveTwitchToken(ctx, botToken); err != nil {
					return fmt.Errorf("upserting new token: %w", err)
				}
			}
			if err != nil {
				return fmt.Errorf("getting moderated channels: %w", err)
			}

			for _, channel := range moderatedChannels {
				if err := q.UpsertModeratedChannel(ctx, dbsql.UpsertModeratedChannelParams{
					BotName:          botName,
					BroadcasterID:    int64(channel.ID),
					BroadcasterLogin: channel.Login,
					BroadcasterName:  channel.Name,
					UpdatedAt:        dbsql.TimestamptzFrom(start),
				}); err != nil {
					return fmt.Errorf("upserting moderated channel: %w", err)
				}
			}

			if err := q.DeleteStaleModeratedChannels(ctx, dbsql.DeleteStaleModeratedChannelsParams{
				BotName:       botName,
				UpdatedBefore: dbsql.TimestamptzFrom(start),
			}); err != nil {
				return fmt.Errorf("deleting old moderated channels: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	logFn(ctx, "updated moderated channels",
		zap.Duration("duration", time.Since(start)),
	)
	return nil
}

func (b *Bot) runUpdateModeratedChannels(ctx context.Context) error {
	for {
		log := false
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tickerChan(b.updateModeratedChannelsTicker):
		case <-b.updateModeratedChannelsManual:
			log = true
		}

		if err := b.updateModeratedChannels(ctx, log); err != nil {
			ctxlog.Error(ctx, "failed to update moderated channels", zap.Error(err))
		}
	}
}

func (b *Bot) updateModeratedChannelsNow() {
	select {
	case b.updateModeratedChannelsManual <- struct{}{}:
	default:
	}
}

func tickerChan(t *time.Ticker) <-chan time.Time {
	if t == nil {
		return nil
	}
	return t.C
}
