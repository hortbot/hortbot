package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/dbx"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

func (b *Bot) validateTokens(ctx context.Context, log bool) error {
	logFn := ctxlog.Debug
	if log {
		logFn = ctxlog.Info
	}

	logFn(ctx, "validating twitch tokens")
	start := b.deps.Clock.Now()

	tokens, err := models.TwitchTokens().All(ctx, b.db)
	if err != nil {
		return fmt.Errorf("getting tokens: %w", err)
	}

	validated := 0
	updated := 0
	deleted := 0

	for _, tt := range tokens {
		err := dbx.Transact(ctx, b.db, func(ctx context.Context, tx *sql.Tx) error {
			ctx = ctxlog.With(ctx, zap.Int64("twitch_id", tt.TwitchID))

			ctxlog.Debug(ctx, "validating token")
			token := modelsx.ModelToToken(tt)
			validation, newToken, err := b.deps.Twitch.Validate(ctx, token)
			if err != nil {
				if errors.Is(err, twitch.ErrDeadToken) || errors.Is(err, twitch.ErrNotAuthorized) || errors.Is(err, twitch.ErrNotFound) {
					ctxlog.Info(ctx, "deleting dead token", zap.Error(err))
					if err := tt.Delete(ctx, b.db); err != nil {
						return fmt.Errorf("deleting dead token: %w", err)
					}
					metricDeletedTokens.Inc()
					deleted++
					return nil
				}
				ctxlog.Error(ctx, "failed to validate token", zap.Error(err))
				metricTokenValidationErrors.Inc()
				return nil
			}

			if newToken != nil {
				tt = modelsx.TokenToModel(newToken, tt.TwitchID, tt.BotName, validation.Scopes)
			} else {
				tt.Scopes = validation.Scopes
			}

			ctxlog.Debug(ctx, "token validated", zap.Bool("new_token", newToken != nil), zap.Strings("scopes", tt.Scopes))
			if err := modelsx.UpsertToken(ctx, b.db, tt); err != nil {
				return err
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
		zap.Duration("duration", b.deps.Clock.Since(start)),
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
		case <-b.validateTokensTicker.Chan():
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
	start := b.deps.Clock.Now()

	conflictColumns := []string{
		models.ModeratedChannelColumns.BotName,
		models.ModeratedChannelColumns.BroadcasterID,
	}

	err := dbx.Transact(ctx, b.db, func(ctx context.Context, tx *sql.Tx) error {
		botTokens, err := models.TwitchTokens(
			models.TwitchTokenWhere.BotName.IsNotNull(),
			qm.Where("scopes @> array['user:read:moderated_channels']"),
		).All(ctx, tx)
		if err != nil {
			return fmt.Errorf("getting bot tokens: %w", err)
		}

		logFn(ctx, "locking moderated_channels table")
		if _, err := tx.ExecContext(ctx, "LOCK TABLE moderated_channels IN EXCLUSIVE MODE"); err != nil {
			return fmt.Errorf("locking moderated_channels: %w", err)
		}

		start := b.deps.Clock.Now()

		for _, botToken := range botTokens {
			botName := botToken.BotName.String

			logFn(ctx, "updating bot", zap.String("bot_name", botName))
			token := modelsx.ModelToToken(botToken)
			moderatedChannels, newToken, err := b.deps.Twitch.GetModeratedChannels(ctx, botToken.TwitchID, token)
			if newToken != nil {
				botToken := modelsx.TokenToModel(newToken, botToken.TwitchID, botToken.BotName, botToken.Scopes)
				if err := modelsx.UpsertToken(ctx, tx, botToken); err != nil {
					return err
				}
			}
			if err != nil {
				return fmt.Errorf("getting moderated channels: %w", err)
			}

			for _, channel := range moderatedChannels {
				m := &models.ModeratedChannel{
					BotName:          botName,
					BroadcasterID:    int64(channel.ID),
					BroadcasterLogin: channel.Login,
					BroadcasterName:  channel.Name,
				}

				if err := m.Upsert(ctx, tx, true, conflictColumns, boil.Blacklist(models.ModeratedChannelColumns.CreatedAt), boil.Infer()); err != nil {
					return fmt.Errorf("upserting moderated channel: %w", err)
				}
			}

			if err := models.ModeratedChannels(
				models.ModeratedChannelWhere.BotName.EQ(botName),
				models.ModeratedChannelWhere.UpdatedAt.LT(start),
			).DeleteAll(ctx, tx); err != nil {
				return fmt.Errorf("deleting old moderated channels: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	logFn(ctx, "updated moderated channels",
		zap.Duration("duration", b.deps.Clock.Since(start)),
	)
	return nil
}

func (b *Bot) runUpdateModeratedChannels(ctx context.Context) error {
	for {
		log := false
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-b.updateModeratedChannelsTicker.Chan():
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
