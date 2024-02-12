package bot

import (
	"context"
	"database/sql"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/dbx"
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
		return err
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
				if err == twitch.ErrDeadToken || err == twitch.ErrNotAuthorized || err == twitch.ErrNotFound {
					ctxlog.Info(ctx, "deleting dead token", zap.Error(err))
					if err := tt.Delete(ctx, b.db); err != nil {
						return err
					}
					metricDeletedTokens.Inc()
					deleted++
					return nil
				}
				ctxlog.Error(ctx, "failed to validate token", zap.Error(err))
				metricTokenValidationErrors.Inc()
				return nil
			}

			botName := tt.BotName
			if newToken != nil {
				tt = modelsx.TokenToModel(tt.TwitchID, newToken)
				tt.BotName = botName
			}
			tt.Scopes = validation.Scopes

			ctxlog.Debug(ctx, "token validated", zap.Bool("new_token", newToken != nil), zap.Strings("scopes", tt.Scopes))
			if err := modelsx.FullUpsertToken(ctx, b.db, tt); err != nil {
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
