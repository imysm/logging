package logging

import (
	"context"
	"log"
	"os"
)

type ctxKey string

const traceIDKey ctxKey = "trace_id"

// Init initializes a basic logger that writes to stderr with standard flags.
// Call once on startup.
func Init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags)
}

// WithTraceID returns a child context carrying a trace ID for downstream logging.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceID extracts the trace ID from context if present.
func TraceID(ctx context.Context) string {
	v := ctx.Value(traceIDKey)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
