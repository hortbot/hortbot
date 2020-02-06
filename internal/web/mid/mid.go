// Package mid provides HTTP middleware.
package mid

import (
	"context"
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/rs/xid"
	"go.opencensus.io/plugin/ochttp"
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

// RequestID ensures that a request ID exists on the request and is propogated
// to logging and the outgoing response.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var id xid.ID
		requestID := r.Header.Get(requestIDHeader)

		if requestID != "" {
			var err error
			id, err = xid.FromString(requestID)
			if err != nil {
				oldRequestID := requestID
				id = xid.New()
				requestID = id.String()

				ctxlog.Debug(ctx, "replacing request ID", zap.String("old", oldRequestID), zap.String("new", requestID))
			}
		} else {
			id = xid.New()
			requestID = id.String()
		}

		w.Header().Set(requestIDHeader, requestID)
		ctx = context.WithValue(ctx, requestIDKey{}, id)
		ctx = ctxlog.With(ctx, zap.String("requestID", requestID))

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// GetRequestID gets the request ID for a given request.
func GetRequestID(r *http.Request) xid.ID {
	requestID, _ := r.Context().Value(requestIDKey{}).(xid.ID)
	return requestID
}

// RequestLogger logs information about the request, including the method,
// URL, status, size, and handle duration.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := httpsnoop.CaptureMetrics(next, w, r)

		ctxlog.Debug(r.Context(), "http request",
			zap.String("method", r.Method),
			zap.String("url", r.RequestURI),
			zap.String("proto", r.Proto),
			zap.Int("status", m.Code),
			zap.Int64("size", m.Written),
			zap.Duration("duration", m.Duration),
		)
	})
}

// Recoverer recovers from panics, writing out an HTTP error message when needed.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				// Ensure logger is logging stack traces, at least here.
				ctx := ctxlog.WithOptions(r.Context(), zap.AddStacktrace(zap.ErrorLevel))
				ctxlog.Error(ctx, "PANIC", zap.Any("panic_value", rvr))

				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// Tracer traces the handler with an OpenCensus HTTP tracer.
func Tracer(next http.Handler) http.Handler {
	return &ochttp.Handler{
		Handler: next,
	}
}
