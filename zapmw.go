package zapmw

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
)

type zapmwkey int

const key zapmwkey = iota

// New returns a new logging middleware using the provided *zap.Logger
func New(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// set incomplete request fields
			logger = logger.With(
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
				zap.Time("start_time", start),
			)

			// wrap response writer
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// store logger in context
			ctx := context.WithValue(r.Context(), key, logger)

			// invoke next handler
			next.ServeHTTP(ww, r.WithContext(ctx))

			// get completed request fields
			status := ww.Status()
			logger = logger.With(
				zap.Duration("duration", time.Since(start)),
				zap.Int("status", status),
				zap.Int("bytes_written", ww.BytesWritten()),
			)

			switch {
			case status < 400:
				logger.Info("request complete")
			default:
				logger.Error("request completed with error")
			}
		})
	}
}

// Extract returns the *zap.Logger set by zapmw. If no logger is
// found in the context, a no op logger is returned.
func Extract(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(key).(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop()
}
