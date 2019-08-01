package mid

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/rs/xid"
	"go.uber.org/zap"
)

type requestIDKey struct{}

const requestIDHeader = "X-Request-ID"

// Logger adds a logger to a Handler chain.
func Logger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := ctxlog.WithLogger(r.Context(), logger)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := ctxlog.FromContext(ctx)

		var id xid.ID
		requestID := r.Header.Get(requestIDHeader)

		if requestID != "" {
			var err error
			id, err = xid.FromString(requestID)
			if err != nil {
				oldRequestID := requestID
				id = xid.New()
				requestID = id.String()

				logger.Debug("replacing request ID", zap.String("old", oldRequestID), zap.String("new", requestID))
			}
		} else {
			id = xid.New()
			requestID = id.String()
		}

		w.Header().Set(requestIDHeader, requestID)
		ctx = context.WithValue(ctx, requestIDKey{}, id)

		ctx, _ = ctxlog.FromContextWith(ctx, zap.String("requestID", requestID))

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func GetRequestID(r *http.Request) xid.ID {
	requestID, _ := r.Context().Value(requestIDKey{}).(xid.ID)
	return requestID
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()

		defer func() {
			duration := time.Since(start)
			logger := ctxlog.FromContext(r.Context())

			logger.Debug("http request",
				zap.String("method", r.Method),
				zap.String("url", r.RequestURI),
				zap.String("proto", r.Proto),
				zap.Int("status", ww.Status()),
				zap.Int("size", ww.BytesWritten()),
				zap.Duration("duration", duration),
			)
		}()

		next.ServeHTTP(ww, r)
	})
}

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				logger := ctxlog.FromContext(r.Context())

				// Ensure logger is logging stack traces, at least here.
				logger = logger.WithOptions(zap.AddStacktrace(zap.ErrorLevel))

				logger.Error("PANIC",
					zap.Any("panic_value", rvr),
				)

				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
