# Logging

A simple, structured logging library for Go applications built on top of Go 1.21+ `log/slog`. This library provides an easy-to-use interface for logging with support for log rotation, structured fields, context propagation, and multiple output formats.

## Features

- **Five Log Levels**: `Trace`, `Debug`, `Info`, `Warn`, `Error` — each level supports three variants: basic, `WithFields`, and `WithCtx`
- **Context Propagation**: Automatic trace ID propagation through `context.Context` via `WithTraceID` / `TraceID`
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
go get github.com/samyang/logging
```

## Quick Start

```go
package main

import (
    "github.com/samyang/logging"
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
| AlertPretty | bool   | Whether to pretty-print alert logs                                         | false        |

## Usage Examples

### Basic Logging

```go
logging.Logger.Trace("Very detailed trace information")
logging.Logger.Debug("Detailed debug information")
logging.Logger.Info("General information")
logging.Logger.Warn("Warning message")
logging.Logger.Error("Error occurred")
```

### Printf-Style Formatting

```go
logging.Logger.Info("User %s performed %d actions", "Alice", 42)
```

### Structured Logging with Fields

```go
logging.Logger.InfoWithFields("Database query", map[string]interface{}{
    "query": "SELECT * FROM users WHERE id = ?",
    "duration_ms": 15,
    "rows": 10,
})
```

### Context-Aware Logging

```go
import "context"

// Add trace ID to context
ctx := logging.WithTraceID(context.Background(), "trace-abc-123")

// Option 1: Use WithCtx methods
logging.Logger.InfoWithCtx(ctx, "Processing request")

// Option 2: Use L(ctx) for a context-bound logger
log := logging.L(ctx)
log.Info("Processing request")
log.Warn("Slow query detected")
log.ErrorWithFields("Request failed", map[string]interface{}{
    "code": 500,
    "path": "/api/users",
})
```

### Dynamic Level Control

```go
// Change log level at runtime
logging.Logger.SetLevel(logging.LevelTrace)

// All levels including trace will now be logged
logging.Logger.Debug("This will be visible")
```

### Creating Separate Log Files

```go
// Create a separate access log with the same rotation policy
accessLog := logging.GetRotatedWriter("log/access.log")
defer accessLog.Close()

accessLog.Write([]byte("GET /api/users 200\n"))
```

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

This library is built for performance. Benchmarks show:

```
BenchmarkLoggerInfo-8          1000000    1023 ns/op
BenchmarkLoggerWithFields-8     500000    2156 ns/op
```

The library minimizes allocations and uses efficient buffering for log rotation.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built on top of Go 1.21+ `log/slog`
- Uses [lumberjack](https://github.com/natefinch/lumberjack) for log rotation
