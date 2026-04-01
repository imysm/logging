package logging

import (
	"context"
	"log"
	"os"
)

type ctxKey string

const traceIDKey ctxKey = "trace_id"
const ctxFieldsKey ctxKey = "ctx_fields"

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

// WithCtxFields returns a child context carrying custom fields for downstream logging.
// These fields will be automatically included in all log entries when using L(ctx).
//
//	ctx = logging.WithCtxFields(ctx, map[string]interface{}{
//	    "user_id":    123,
//	    "request_id": "req-abc",
//	})
//	logging.L(ctx).Info("processing") // logs include user_id and request_id
func WithCtxFields(ctx context.Context, fields map[string]interface{}) context.Context {
	existing := CtxFields(ctx)
	merged := make(map[string]interface{}, len(existing)+len(fields))
	for k, v := range existing {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}
	return context.WithValue(ctx, ctxFieldsKey, merged)
}

// CtxFields extracts custom fields from context if present.
func CtxFields(ctx context.Context) map[string]interface{} {
	v := ctx.Value(ctxFieldsKey)
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// L returns a context-bound logger. All log methods on the returned logger
// will automatically include trace_id and other context values.
//
//	log.L(ctx).Info("request processed")
//	log.L(ctx).InfoWithFields("user login", map[string]interface{}{"user_id": 1})
func L(ctx context.Context) *ContextLogger {
	return &ContextLogger{ctx: ctx}
}

// ContextLogger is a logger bound to a context.Context.
// It implements LoggerInterface and can be used as a drop-in replacement.
type ContextLogger struct {
	ctx context.Context
}

func (c *ContextLogger) Trace(format string, v ...interface{}) {
	Logger.TraceWithCtx(c.ctx, format, v...)
}

func (c *ContextLogger) Debug(format string, v ...interface{}) {
	Logger.DebugWithCtx(c.ctx, format, v...)
}

func (c *ContextLogger) Info(format string, v ...interface{}) {
	Logger.InfoWithCtx(c.ctx, format, v...)
}

func (c *ContextLogger) Warn(format string, v ...interface{}) {
	Logger.WarnWithCtx(c.ctx, format, v...)
}

func (c *ContextLogger) Error(format string, v ...interface{}) {
	Logger.ErrorWithCtx(c.ctx, format, v...)
}

func (c *ContextLogger) TraceWithFields(format string, fields map[string]interface{}, v ...interface{}) {
	Logger.TraceWithFields(format, c.mergedFields(fields), v...)
}

func (c *ContextLogger) DebugWithFields(format string, fields map[string]interface{}, v ...interface{}) {
	Logger.DebugWithFields(format, c.mergedFields(fields), v...)
}

func (c *ContextLogger) InfoWithFields(format string, fields map[string]interface{}, v ...interface{}) {
	Logger.InfoWithFields(format, c.mergedFields(fields), v...)
}

func (c *ContextLogger) WarnWithFields(format string, fields map[string]interface{}, v ...interface{}) {
	Logger.WarnWithFields(format, c.mergedFields(fields), v...)
}

func (c *ContextLogger) ErrorWithFields(format string, fields map[string]interface{}, v ...interface{}) {
	Logger.ErrorWithFields(format, c.mergedFields(fields), v...)
}

func (c *ContextLogger) TraceWithCtx(ctx context.Context, format string, v ...interface{}) {
	Logger.TraceWithCtx(ctx, format, v...)
}

func (c *ContextLogger) DebugWithCtx(ctx context.Context, format string, v ...interface{}) {
	Logger.DebugWithCtx(ctx, format, v...)
}

func (c *ContextLogger) InfoWithCtx(ctx context.Context, format string, v ...interface{}) {
	Logger.InfoWithCtx(ctx, format, v...)
}

func (c *ContextLogger) WarnWithCtx(ctx context.Context, format string, v ...interface{}) {
	Logger.WarnWithCtx(ctx, format, v...)
}

func (c *ContextLogger) ErrorWithCtx(ctx context.Context, format string, v ...interface{}) {
	Logger.ErrorWithCtx(ctx, format, v...)
}

func (c *ContextLogger) SetLevel(level LogLevel) {
	Logger.SetLevel(level)
}

func (c *ContextLogger) Sync() error {
	return Logger.Sync()
}

// mergedFields returns a copy of fields with context fields (trace_id and custom ctx fields) injected.
func (c *ContextLogger) mergedFields(fields map[string]interface{}) map[string]interface{} {
	ctxFields := CtxFields(c.ctx)
	traceID := TraceID(c.ctx)
	if len(ctxFields) == 0 && traceID == "" {
		return fields
	}
	merged := make(map[string]interface{}, len(fields)+len(ctxFields)+1)
	for k, v := range ctxFields {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}
	if traceID != "" {
		merged["trace_id"] = traceID
	}
	return merged
}
