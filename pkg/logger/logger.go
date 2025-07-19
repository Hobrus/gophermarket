package logger

import (
	"context"
	"net/http"
	"os"

	"github.com/rs/zerolog"
)

// ctxKey is the type used for storing values in context
// to avoid collisions.
type ctxKey string

const (
	loggerKey    ctxKey = "logger"
	requestIDKey ctxKey = "request_id"
)

// Init configures zerolog with the provided level and returns
// a base logger instance.
func Init(level string) *zerolog.Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	l := zerolog.New(os.Stdout).Level(lvl).With().Timestamp().Logger()
	return &l
}

// FromContext returns logger from context if present, otherwise
// returns a default logger.
func FromContext(ctx context.Context) *zerolog.Logger {
	if ctx == nil {
		return nil
	}
	if l, ok := ctx.Value(loggerKey).(*zerolog.Logger); ok {
		return l
	}
	return nil
}

// Middleware injects a logger into request context. Logger will
// contain request_id field if it's stored in context by previous middleware.
func Middleware(base *zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract request ID from context if available.
			reqID, _ := r.Context().Value(requestIDKey).(string)
			logger := base.With().Str(string(requestIDKey), reqID).Logger()
			ctx := context.WithValue(r.Context(), loggerKey, &logger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// WithRequestID stores request ID in context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext retrieves request ID.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}
