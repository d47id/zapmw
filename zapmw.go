package zapmw

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapmwkey int

const key zapmwkey = iota

type option struct {
	setter func(*options)
}

func (o option) set(opts *options) {
	o.setter(opts)
}

type options struct {
	SuccessLevel     zapcore.Level
	RedirectionLevel zapcore.Level
	ClientErrorLevel zapcore.Level
	ServerErrorLevel zapcore.Level
}

// Option configures log levels for response codes.
type Option interface {
	set(opts *options)
}

// WithSuccessLevel sets the log level for 2xx responses. The default is DebugLevel.
func WithSuccessLevel(level zapcore.Level) Option {
	return option{
		setter: func(opts *options) {
			opts.SuccessLevel = level
		},
	}
}

// WithRedirectionLevel sets the log level for 3xx responses. The default is DebugLevel.
func WithRedirectionLevel(level zapcore.Level) Option {
	return option{
		setter: func(opts *options) {
			opts.RedirectionLevel = level
		},
	}
}

// WithClientErrorLevel sets the log level for 4xx responses. The default is DebugLevel.
func WithClientErrorLevel(level zapcore.Level) Option {
	return option{
		setter: func(opts *options) {
			opts.ClientErrorLevel = level
		},
	}
}

// WithServerErrorLevel sets the log level for 5xx responses. The default is ErrorLevel.
func WithServerErrorLevel(level zapcore.Level) Option {
	return option{
		setter: func(opts *options) {
			opts.ServerErrorLevel = level
		},
	}
}

// New returns a new logging middleware using the provided *zap.Logger
func New(logger *zap.Logger, opts ...Option) func(next http.Handler) http.Handler {
	// default log levels
	defaultOpts := &options{
		SuccessLevel:     zapcore.DebugLevel,
		RedirectionLevel: zapcore.DebugLevel,
		ClientErrorLevel: zapcore.DebugLevel,
		ServerErrorLevel: zapcore.ErrorLevel,
	}

	// apply options
	for _, o := range opts {
		o.set(defaultOpts)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// set incomplete request fields
			l := logger.With(
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
				zap.String("referrer", r.Referer()),
				zap.Time("start_time", start),
			)

			// wrap response writer
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// store logger in context
			ctx := context.WithValue(r.Context(), key, l)

			// invoke next handler
			next.ServeHTTP(ww, r.WithContext(ctx))

			// get completed request fields
			status := ww.Status()
			l = l.With(
				zap.Duration("duration", time.Since(start)),
				zap.Int("status", status),
				zap.Int("bytes_written", ww.BytesWritten()),
			)

			logHTTPStatus(l, defaultOpts, status)
		})
	}
}

func logHTTPStatus(l *zap.Logger, opts *options, status int) {
	var msg string
	if msg = http.StatusText(status); msg == "" {
		msg = "unknown status " + strconv.Itoa(status)
	}

	var level zapcore.Level
	switch {
	case status >= 500:
		level = opts.ServerErrorLevel
	case status >= 400:
		level = opts.ClientErrorLevel
	case status >= 300:
		level = opts.RedirectionLevel
	default:
		level = opts.SuccessLevel
	}

	if ce := l.Check(level, msg); ce != nil {
		ce.Write()
	}
}

// Extract returns the *zap.Logger set by zapmw. If no logger is
// found in the context, zap.NewNop() is returned.
func Extract(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(key).(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop()
}
