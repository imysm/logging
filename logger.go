package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"log/slog"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	// DefaultLogLevel is the default log level if not specified
	DefaultLogLevel = "info"
	// DefaultLogFile is the default log file path
	DefaultLogFile = "log/app.log"
	// DefaultLogFormat is the default log format
	DefaultLogFormat = "text"
	// DefaultLogOutput is the default output destination
	DefaultLogOutput = "both"
)

// LoggerInterface defines the logging abstraction for easy mocking in tests.
type LoggerInterface interface {
	Trace(format string, v ...interface{})
	Debug(format string, v ...interface{})
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
	TraceWithFields(format string, fields map[string]interface{}, v ...interface{})
	DebugWithFields(format string, fields map[string]interface{}, v ...interface{})
	InfoWithFields(format string, fields map[string]interface{}, v ...interface{})
	WarnWithFields(format string, fields map[string]interface{}, v ...interface{})
	ErrorWithFields(format string, fields map[string]interface{}, v ...interface{})
	TraceWithCtx(ctx context.Context, format string, v ...interface{})
	DebugWithCtx(ctx context.Context, format string, v ...interface{})
	InfoWithCtx(ctx context.Context, format string, v ...interface{})
	WarnWithCtx(ctx context.Context, format string, v ...interface{})
	ErrorWithCtx(ctx context.Context, format string, v ...interface{})
	SetLevel(level LogLevel)
	Sync() error
}

// LogLevel represents the severity level of a log entry.
type LogLevel int

const (
	LevelTrace LogLevel = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
)

// slogLevelTrace is the slog level for trace, lower than slog.LevelDebug (-4).
const slogLevelTrace slog.Level = -8

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case LevelTrace:
		return "TRACE"
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger is the global logger instance. The interface type allows for test replacement.
var Logger LoggerInterface

var (
	loggerInitMu sync.Mutex
	loggerOnce   sync.Once
)

// globalLogConfig stores the initialization configuration for creating rotated log files.
var globalLogConfig LogConfig

// LogConfig defines the configuration for the logger.
type LogConfig struct {
	Level       string // Log level: trace, debug, info, warn, error
	File        string // Log file path
	MaxSize     int    // Maximum size in megabytes before rotation
	MaxBackups  int    // Maximum number of old log files to retain
	MaxAge      int    // Maximum number of days to retain old log files
	Compress    bool   // Whether to compress rotated log files
	Format      string // Log format: "text" or "json"
	Outputs     string // Output destination: "console", "file", "both"
	AlertPretty bool   // Whether to pretty-print alert logs
}

// Validate checks if the configuration is valid and returns a normalized config.
func (c *LogConfig) Validate() error {
	if c.Level == "" {
		c.Level = DefaultLogLevel
	}
	if c.File == "" {
		c.File = DefaultLogFile
	}
	if c.MaxSize <= 0 {
		c.MaxSize = 10 // Default 10MB
	}
	if c.MaxBackups <= 0 {
		c.MaxBackups = 5 // Default keep 5 backups
	}
	if c.MaxAge <= 0 {
		c.MaxAge = 30 // Default 30 days
	}
	if c.Format == "" {
		c.Format = DefaultLogFormat
	}
	if c.Outputs == "" {
		c.Outputs = DefaultLogOutput
	}

	// Validate log level
	switch strings.ToLower(c.Level) {
	case "trace", "debug", "info", "warn", "error", "warning":
		// Valid
	default:
		return fmt.Errorf("invalid log level: %s", c.Level)
	}

	// Validate format
	switch c.Format {
	case "text", "json":
		// Valid
	default:
		return fmt.Errorf("invalid log format: %s (must be 'text' or 'json')", c.Format)
	}

	// Validate outputs
	switch strings.ToLower(c.Outputs) {
	case "console", "file", "both":
		// Valid
	default:
		return fmt.Errorf("invalid log output: %s (must be 'console', 'file', or 'both')", c.Outputs)
	}

	return nil
}

type logger struct {
	lg     *slog.Logger
	level  LogLevel
	writer io.Writer
	format string // "text" or "json"
}

// InitLogger initializes the logger with the provided configuration.
// Call this in main after loading the configuration.
func InitLogger(cfg LogConfig) {
	loggerInitMu.Lock()
	defer loggerInitMu.Unlock()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid log configuration: %v\n", err)
		cfg = LogConfig{
			Level:  DefaultLogLevel,
			File:   DefaultLogFile,
			Format: DefaultLogFormat,
			Outputs: DefaultLogOutput,
		}
		cfg.Validate()
	}

	globalLogConfig = cfg

	lvl := parseLogLevel(cfg.Level)
	l := &logger{
		level:  lvl,
		format: cfg.Format,
	}

	// Ensure log directory exists
	dir := filepath.Dir(cfg.File)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			// If directory creation fails, fall back to stdout
			fmt.Fprintf(os.Stderr, "Failed to create log directory: %v, falling back to stdout\n", err)
			cfg.File = ""
			cfg.Outputs = "console"
		}
	}

	// Setup output writers based on cfg.Outputs
	var writers []io.Writer
	var fileWriter *lumberjack.Logger

	if cfg.File != "" {
		// File output writer
		fileWriter = &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSize,    // megabytes
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,     // days
			Compress:   cfg.Compress,
		}
	}

	outputs := strings.ToLower(strings.TrimSpace(cfg.Outputs))
	switch outputs {
	case "console":
		writers = append(writers, os.Stdout)
	case "file":
		if fileWriter != nil {
			writers = append(writers, fileWriter)
		} else {
			fmt.Fprintf(os.Stderr, "File output requested but no file configured, falling back to stdout\n")
			writers = append(writers, os.Stdout)
		}
	case "both":
		writers = append(writers, os.Stdout)
		if fileWriter != nil {
			writers = append(writers, fileWriter)
		}
	default:
		// Unknown value falls back to both
		fmt.Fprintf(os.Stderr, "Unknown log.outputs value '%s', falling back to 'both'\n", cfg.Outputs)
		writers = append(writers, os.Stdout)
		if fileWriter != nil {
			writers = append(writers, fileWriter)
		}
	}

	if len(writers) == 0 {
		// Fallback to stdout in extreme cases
		writers = append(writers, os.Stdout)
	}

	l.writer = io.MultiWriter(writers...)

	// Use slog Handler, selecting JSON or text based on format
	var handler slog.Handler
	// Don't print source file path in logs (AddSource=false) to avoid leaking local filesystem paths
	opts := slog.HandlerOptions{AddSource: false}
	// Configure handler's minimum level to ensure underlying handler doesn't filter logs below configured level
	switch l.level {
	case LevelTrace:
		opts.Level = slogLevelTrace
	case LevelDebug:
		opts.Level = slog.LevelDebug
	case LevelInfo:
		opts.Level = slog.LevelInfo
	case LevelWarn:
		opts.Level = slog.LevelWarn
	case LevelError:
		opts.Level = slog.LevelError
	default:
		opts.Level = slog.LevelInfo
	}

	if l.format == "json" {
		handler = slog.NewJSONHandler(l.writer, &opts)
	} else {
		handler = slog.NewTextHandler(l.writer, &opts)
	}
	slogLogger := slog.New(handler)
	l.lg = slogLogger

	Logger = l
}

func parseLogLevel(s string) LogLevel {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "trace":
		return LevelTrace
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// log is a helper method to reduce code duplication in logging methods
func (l *logger) log(level LogLevel, format string, v []interface{}, attrs []slog.Attr) {
	if l.level > level {
		return
	}

	msg := fmt.Sprintf(format, v...)
	allAttrs := append([]slog.Attr{sourceAttr()}, attrs...)

	switch level {
	case LevelTrace:
		l.lg.Log(context.Background(), slogLevelTrace, msg, attrsToAny(allAttrs)...)
	case LevelDebug:
		l.lg.Debug(msg, attrsToAny(allAttrs)...)
	case LevelInfo:
		l.lg.Info(msg, attrsToAny(allAttrs)...)
	case LevelWarn:
		l.lg.Warn(msg, attrsToAny(allAttrs)...)
	case LevelError:
		l.lg.Error(msg, attrsToAny(allAttrs)...)
	}
}

func (l *logger) Trace(format string, v ...interface{}) {
	l.log(LevelTrace, format, v, nil)
}

func (l *logger) TraceWithFields(format string, fields map[string]interface{}, v ...interface{}) {
	merged := WithBaseFields(fields)
	l.log(LevelTrace, format, v, toAttrs(merged))
}

func (l *logger) Debug(format string, v ...interface{}) {
	l.log(LevelDebug, format, v, nil)
}

func (l *logger) DebugWithFields(format string, fields map[string]interface{}, v ...interface{}) {
	merged := WithBaseFields(fields)
	l.log(LevelDebug, format, v, toAttrs(merged))
}

func (l *logger) Info(format string, v ...interface{}) {
	l.log(LevelInfo, format, v, nil)
}

func (l *logger) InfoWithFields(format string, fields map[string]interface{}, v ...interface{}) {
	merged := WithBaseFields(fields)
	l.log(LevelInfo, format, v, toAttrs(merged))
}

func (l *logger) Warn(format string, v ...interface{}) {
	l.log(LevelWarn, format, v, nil)
}

func (l *logger) WarnWithFields(format string, fields map[string]interface{}, v ...interface{}) {
	merged := WithBaseFields(fields)
	l.log(LevelWarn, format, v, toAttrs(merged))
}

func (l *logger) Error(format string, v ...interface{}) {
	l.log(LevelError, format, v, nil)
}

func (l *logger) ErrorWithFields(format string, fields map[string]interface{}, v ...interface{}) {
	merged := WithBaseFields(fields)
	l.log(LevelError, format, v, toAttrs(merged))
}

func (l *logger) SetLevel(level LogLevel) {
	l.level = level
	// Update the slog handler's level as well
	var slogLevel slog.Level
	switch level {
	case LevelTrace:
		slogLevel = slogLevelTrace
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	// Create a new handler with the updated level
	var handler slog.Handler
	opts := slog.HandlerOptions{AddSource: false, Level: slogLevel}
	if l.format == "json" {
		handler = slog.NewJSONHandler(l.writer, &opts)
	} else {
		handler = slog.NewTextHandler(l.writer, &opts)
	}
	l.lg = slog.New(handler)
}

// Sync flushes any buffered log entries. Implements LoggerInterface.
func (l *logger) Sync() error {
	if syncer, ok := l.writer.(interface{ Sync() error }); ok {
		return syncer.Sync()
	}
	return nil
}

// logWithCtx is a helper method for context-aware logging
func (l *logger) logWithCtx(ctx context.Context, level LogLevel, format string, v []interface{}) {
	if l.level > level {
		return
	}

	msg := fmt.Sprintf(format, v...)
	attrs := []slog.Attr{sourceAttr()}

	if traceID := TraceID(ctx); traceID != "" {
		attrs = append(attrs, slog.String("trace_id", traceID))
	}

	switch level {
	case LevelTrace:
		l.lg.Log(context.Background(), slogLevelTrace, msg, attrsToAny(attrs)...)
	case LevelDebug:
		l.lg.Debug(msg, attrsToAny(attrs)...)
	case LevelInfo:
		l.lg.Info(msg, attrsToAny(attrs)...)
	case LevelWarn:
		l.lg.Warn(msg, attrsToAny(attrs)...)
	case LevelError:
		l.lg.Error(msg, attrsToAny(attrs)...)
	}
}

// TraceWithCtx logs with context, automatically extracting trace_id if present
func (l *logger) TraceWithCtx(ctx context.Context, format string, v ...interface{}) {
	l.logWithCtx(ctx, LevelTrace, format, v)
}

// DebugWithCtx logs with context, automatically extracting trace_id if present
func (l *logger) DebugWithCtx(ctx context.Context, format string, v ...interface{}) {
	l.logWithCtx(ctx, LevelDebug, format, v)
}

// InfoWithCtx logs with context, automatically extracting trace_id if present
func (l *logger) InfoWithCtx(ctx context.Context, format string, v ...interface{}) {
	l.logWithCtx(ctx, LevelInfo, format, v)
}

// WarnWithCtx logs with context, automatically extracting trace_id if present
func (l *logger) WarnWithCtx(ctx context.Context, format string, v ...interface{}) {
	l.logWithCtx(ctx, LevelWarn, format, v)
}

// ErrorWithCtx logs with context, automatically extracting trace_id if present
func (l *logger) ErrorWithCtx(ctx context.Context, format string, v ...interface{}) {
	l.logWithCtx(ctx, LevelError, format, v)
}

func toAttrs(fields map[string]interface{}) []slog.Attr {
	if fields == nil {
		return nil
	}
	attrs := make([]slog.Attr, 0, len(fields))
	for k, v := range fields {
		switch t := v.(type) {
		case string:
			attrs = append(attrs, slog.String(k, t))
		case int:
			attrs = append(attrs, slog.Int(k, t))
		case int64:
			attrs = append(attrs, slog.Int64(k, t))
		case float64:
			attrs = append(attrs, slog.Float64(k, t))
		case bool:
			attrs = append(attrs, slog.Bool(k, t))
		default:
			attrs = append(attrs, slog.Any(k, v))
		}
	}
	return attrs
}

// shouldSkipFrame determines if a frame should be skipped when finding the call site.
func shouldSkipFrame(function, filename string) bool {
	// Skip frames from this logger implementation
	if filename == "logger.go" {
		return true
	}
	// Skip testing and runtime frames
	if strings.HasPrefix(function, "testing.") ||
		strings.Contains(function, "testing.tRunner") ||
		strings.HasPrefix(function, "runtime.") {
		return true
	}
	return false
}

func sourceAttr() slog.Attr {
	// Walk the call stack and return the first frame that is outside
	// the logger implementation. This yields the call-site in user code.
	pcs := make([]uintptr, 32)
	n := runtime.Callers(3, pcs)
	if n == 0 {
		return slog.String("source", "unknown")
	}
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		base := filepath.Base(frame.File)

		if shouldSkipFrame(frame.Function, base) {
			if !more {
				break
			}
			continue
		}

		// Extract a short function name (last element after '/') then last part after '.'
		funcName := frame.Function
		if i := strings.LastIndex(funcName, "/"); i != -1 {
			funcName = funcName[i+1:]
		}
		if i := strings.LastIndex(funcName, "."); i != -1 {
			funcName = funcName[i+1:]
		}
		return slog.String("source", fmt.Sprintf("%s %s:%d", funcName, base, frame.Line))
	}
	return slog.String("source", "unknown")
}

func attrsToAny(a []slog.Attr) []any {
	if a == nil {
		return nil
	}
	out := make([]any, len(a))
	for i := range a {
		out[i] = a[i]
	}
	return out
}

// ---- Base fields management ----

var (
	baseFieldsMu sync.RWMutex
	baseFields   map[string]interface{}
)

// SetBaseFields sets global base fields, typically called once at program startup.
func SetBaseFields(fields map[string]interface{}) {
	baseFieldsMu.Lock()
	defer baseFieldsMu.Unlock()
	if fields == nil {
		baseFields = nil
		return
	}
	bf := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		bf[k] = v
	}
	baseFields = bf
}

// WithBaseFields returns a new map merging global base fields with the provided fields.
func WithBaseFields(fields map[string]interface{}) map[string]interface{} {
	baseFieldsMu.RLock()
	defer baseFieldsMu.RUnlock()
	if baseFields == nil && fields == nil {
		return nil
	}
	var out map[string]interface{}
	if baseFields == nil {
		out = make(map[string]interface{}, len(fields))
	} else {
		out = make(map[string]interface{}, len(baseFields)+len(fields))
		for k, v := range baseFields {
			out[k] = v
		}
	}
	for k, v := range fields {
		out[k] = v
	}
	return out
}

// GetBaseFields returns a copy of the currently set global base fields.
func GetBaseFields() map[string]interface{} {
	baseFieldsMu.RLock()
	defer baseFieldsMu.RUnlock()
	if baseFields == nil {
		return nil
	}
	copy := make(map[string]interface{}, len(baseFields))
	for k, v := range baseFields {
		copy[k] = v
	}
	return copy
}

// GetGlobalConfig returns a copy of the global log configuration.
func GetGlobalConfig() LogConfig {
	return globalLogConfig
}

// GetRotatedWriter returns a rotated io.WriteCloser using the global configuration's rotation policy.
func GetRotatedWriter(filename string) io.WriteCloser {
	cfg := globalLogConfig
	maxSize := cfg.MaxSize
	if maxSize <= 0 {
		maxSize = 10
	}
	maxBackups := cfg.MaxBackups
	if maxBackups <= 0 {
		maxBackups = 5
	}
	maxAge := cfg.MaxAge
	if maxAge <= 0 {
		maxAge = 30
	}

	return &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxSize, // megabytes
		MaxBackups: maxBackups,
		MaxAge:     maxAge, // days
		Compress:   cfg.Compress,
	}
}
