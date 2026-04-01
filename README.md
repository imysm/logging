# Logging

A simple, structured logging library for Go applications built on top of Go 1.21+ `log/slog`. This library provides an easy-to-use interface for logging with support for log rotation, structured fields, context propagation, and multiple output formats.

## Features

- **Five Log Levels**: `Trace`, `Debug`, `Info`, `Warn`, `Error` — each level supports three variants: basic, `WithFields`, and `WithCtx`
- **Context Propagation**: Automatic trace ID propagation through `context.Context` via `WithTraceID` / `TraceID`
- **Context-Bound Logger**: `L(ctx)` returns a logger that automatically carries context values, no need to pass ctx on each call
- **Chain Field Propagation**: `WithCtxFields(ctx, fields)` stores custom fields in context that flow through the entire call chain
- **Structured Logging**: Log with custom fields (`WithFields`) for better searchability and analysis
- **Log Rotation**: Automatic log file rotation with size, age, and backup limits using [lumberjack](https://github.com/natefinch/lumberjack)
- **Multiple Formats**: Text and JSON output formats, powered by `log/slog`
- **Flexible Output**: Log to `console`, `file`, or `both` simultaneously
- **Global Base Fields**: Set base fields via `SetBaseFields` that automatically appear in all log entries
- **Call Source**: Every log entry automatically includes the caller's function name, file, and line number
- **Dynamic Level Control**: Change log level at runtime via `SetLevel`
- **Separate Log Files**: Create independent rotated writers via `GetRotatedWriter` for access logs, audit logs, etc.
- **Mock Logger**: Built-in `MockLogger` for testing with assertion helpers (`HasEntry`, `LastEntry`, `LastStructuredEntry`, `Clear`)
- **Thread-Safe**: Concurrent-safe logging with proper synchronization

## Requirements

- Go 1.21 or higher

## Installation

```bash
go get github.com/imysm/logging
```

## Quick Start

```go
package main

import (
    "github.com/imysm/logging"
)

func main() {
    // Initialize the logger
    logging.InitLogger(logging.LogConfig{
        Level:      "info",
        File:       "log/app.log",
        MaxSize:    100,    // megabytes
        MaxBackups: 5,      // number of backups
        MaxAge:     30,     // days
        Compress:   true,   // compress old logs
        Format:     "text", // or "json"
        Outputs:    "both", // "console", "file", or "both"
    })

    // Set global base fields (optional)
    logging.SetBaseFields(map[string]interface{}{
        "service": "my-api",
        "version": "1.0.0",
    })

    // Use the global logger
    logging.Logger.Info("Application started")
    logging.Logger.InfoWithFields("User logged in", map[string]interface{}{
        "user_id": 12345,
        "ip":      "192.168.1.1",
    })

    // Ensure logs are flushed before exit
    logging.Logger.Sync()
}
```

## Configuration

The `LogConfig` struct allows you to configure the logger:

| Field       | Type    | Description                                                                 | Default      |
|------------|---------|-----------------------------------------------------------------------------|--------------|
| Level      | string  | Log level: "trace", "debug", "info", "warn", "error"                       | "info"       |
| File       | string  | Path to the log file                                                       | "log/app.log" |
| MaxSize    | int     | Maximum size in megabytes before rotation                                  | 10           |
| MaxBackups | int     | Maximum number of old log files to retain                                  | 5            |
| MaxAge     | int     | Maximum number of days to retain old log files                             | 30           |
| Compress   | bool    | Whether to compress rotated log files with gzip                            | false        |
| Format     | string  | Log format: "text" or "json"                                               | "text"       |
| Outputs    | string  | Output destination: "console", "file", or "both"                           | "both"       |

## Usage

### Basic Logging

Each log level has three method variants:

```go
// Basic
logging.Logger.Info("User %s logged in", "Alice")

// With structured fields
logging.Logger.InfoWithFields("User logged in", map[string]interface{}{
    "user_id": 12345,
    "ip":      "192.168.1.1",
})

// With context (auto-includes trace_id and ctx fields)
logging.Logger.InfoWithCtx(ctx, "Processing request")
```

Available for all levels: `Trace`, `Debug`, `Info`, `Warn`, `Error`.

### L(ctx) — Context-Bound Logger

`L(ctx)` returns a logger with context pre-bound. No need to pass ctx on every call:

```go
log := logging.L(ctx)
log.Info("handling request")       // auto-includes ctx fields
log.Warn("slow query")             // auto-includes ctx fields
log.ErrorWithFields("failed", map[string]interface{}{
    "code": 500,
})                                  // auto-includes ctx fields + custom fields
```

`L(ctx)` implements `LoggerInterface`, so it can be used anywhere the global `Logger` is expected.

### Context Field Propagation

`WithCtxFields` stores custom fields in context. They flow through the entire call chain:

```go
// Set fields once at the entry point
ctx = logging.WithCtxFields(ctx, map[string]interface{}{
    "user_id":    123,
    "request_id": "req-xyz",
})

// All downstream functions automatically get these fields
handleRequest(ctx)

func handleRequest(ctx context.Context) {
    log := logging.L(ctx)
    log.Info("handling request")  // includes user_id, request_id
    queryDB(ctx)
}

func queryDB(ctx context.Context) {
    log := logging.L(ctx)
    log.Info("executing query")   // also includes user_id, request_id
}
```

Multiple calls to `WithCtxFields` merge fields (later values overwrite earlier ones):

```go
ctx = logging.WithCtxFields(ctx, map[string]interface{}{"user_id": 123})
ctx = logging.WithCtxFields(ctx, map[string]interface{}{"request_id": "abc"})
// ctx now has both user_id and request_id
```

### Trace ID

```go
ctx := logging.WithTraceID(ctx, "trace-abc-123")
traceID := logging.TraceID(ctx) // "trace-abc-123"
```

### Global Base Fields

Set once at startup, included in all log entries:

```go
logging.SetBaseFields(map[string]interface{}{
    "service": "my-api",
    "version": "1.0.0",
})
defer logging.SetBaseFields(nil)
```

### Dynamic Level Control

```go
logging.Logger.SetLevel(logging.LevelTrace)
logging.Logger.SetLevel(logging.LevelDebug)
logging.Logger.SetLevel(logging.LevelInfo)
logging.Logger.SetLevel(logging.LevelWarn)
logging.Logger.SetLevel(logging.LevelError)
```

### Separate Log Files

Create independent rotated writers for access logs, audit logs, etc.:

```go
accessLog := logging.GetRotatedWriter("log/access.log")
defer accessLog.Close()
accessLog.Write([]byte("GET /api/users 200\n"))
```

Uses the same rotation policy (MaxSize, MaxBackups, MaxAge, Compress) as the main logger.

## API Reference

### Log Levels

| Constant | Value | String |
|----------|-------|--------|
| `logging.LevelTrace` | 0 | `"TRACE"` |
| `logging.LevelDebug` | 1 | `"DEBUG"` |
| `logging.LevelInfo` | 2 | `"INFO"` |
| `logging.LevelWarn` | 3 | `"WARN"` |
| `logging.LevelError` | 4 | `"ERROR"` |

### LoggerInterface

All loggers implement this interface:

```go
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
```

### Package Functions

| Function | Description |
|----------|-------------|
| `InitLogger(cfg LogConfig)` | Initialize the global logger |
| `L(ctx) *ContextLogger` | Return a context-bound logger |
| `WithTraceID(ctx, id) context.Context` | Set trace ID in context |
| `TraceID(ctx) string` | Get trace ID from context |
| `WithCtxFields(ctx, fields) context.Context` | Set custom fields in context |
| `CtxFields(ctx) map[string]interface{}` | Get custom fields from context |
| `SetBaseFields(fields)` | Set global base fields |
| `GetBaseFields() map[string]interface{}` | Get copy of global base fields |
| `WithBaseFields(fields) map[string]interface{}` | Merge global base fields with custom fields |
| `GetGlobalConfig() LogConfig` | Get copy of current config |
| `GetRotatedWriter(filename) io.WriteCloser` | Create a separate rotated log writer |

## Testing

The library includes a `MockLogger` for testing:

```go
func TestMyFunction(t *testing.T) {
    mock := logging.NewMockLogger()
    logging.Logger = mock

    // Set base fields for testing
    logging.SetBaseFields(map[string]interface{}{
        "env": "test",
    })
    defer logging.SetBaseFields(nil)

    // Call your function
    MyFunction()

    // Verify logs
    if !mock.HasEntry("INFO", "expected message") {
        t.Error("expected log entry not found")
    }

    // Clean up for next test
    mock.Clear()
}
```

## Log Levels

The library supports five log levels (in order of severity):

1. **Trace** - Very detailed information for tracing execution flow
2. **Debug** - Detailed information for debugging purposes
3. **Info** - General informational messages
4. **Warn** - Warning messages for potentially harmful situations
5. **Error** - Error messages for error events

When you set a log level, only messages at that level or higher will be logged. For example, if you set the level to "warn", only warn and error messages will be logged.

## Output Formats

### Text Format

```
time=2026-01-10T12:00:00.000+08:00 level=INFO msg="User logged in" source="main.go:42" user_id=12345 ip=192.168.1.1
```

### JSON Format

```json
{
  "time": "2026-01-10T12:00:00.000+08:00",
  "level": "INFO",
  "msg": "User logged in",
  "source": "main.go:42",
  "user_id": 12345,
  "ip": "192.168.1.1"
}
```

## Performance

This library is built for performance. Benchmarks (file output) show:

```
BenchmarkLoggerInfo-8           380000    3295 ns/op    686 B/op    10 allocs/op
BenchmarkLoggerWithFields-8     330000    3650 ns/op   1345 B/op    15 allocs/op
```

The library minimizes allocations and uses efficient buffering for log rotation.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built on top of Go 1.21+ `log/slog`
- Uses [lumberjack](https://github.com/natefinch/lumberjack) for log rotation
