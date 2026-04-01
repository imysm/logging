package logging

import (
	"context"
	"fmt"
	"os"
)

func ExampleInitLogger() {
	// Initialize the logger with configuration
	cfg := LogConfig{
		Level:      "info",
		File:       "log/app.log",
		MaxSize:    100,    // megabytes
		MaxBackups: 5,      // number of backups
		MaxAge:     30,     // days
		Compress:   true,   // compress old logs
		Format:     "text", // or "json"
		Outputs:    "both", // "console", "file", or "both"
	}
	InitLogger(cfg)

	// Now you can use the global Logger
	Logger.Info("Application started")
}

func ExampleLogger_basic() {
	// Basic logging at different levels
	Logger.Trace("This is a trace message")
	Logger.Debug("This is a debug message")
	Logger.Info("This is an info message")
	Logger.Warn("This is a warning message")
	Logger.Error("This is an error message")
}

func ExampleLogger_withFormatting() {
	// Printf-style formatting
	name := "Alice"
	count := 42
	Logger.Info("User %s performed %d actions", name, count)
}

func ExampleLogger_withFields() {
	// Structured logging with custom fields
	Logger.InfoWithFields("User logged in", map[string]interface{}{
		"user_id": 12345,
		"ip":      "192.168.1.1",
		"method":  "oauth",
	})

	// Output (text format):
	// [INFO] User logged in user_id=12345 ip=192.168.1.1 method=oauth

	// Output (json format):
	// {"level":"INFO","time":"...","msg":"User logged in","user_id":12345,"ip":"192.168.1.1","method":"oauth"}
}

func ExampleLogger_withContext() {
	// Create a context with trace ID
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-abc-123")

	// Log with context - trace ID will be included automatically
	Logger.InfoWithCtx(ctx, "Processing request")
	Logger.ErrorWithCtx(ctx, "Request failed")

	// Output includes trace_id field
}

func ExampleL() {
	// Create a context with trace ID
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-abc-123")

	// L returns a context-bound logger, no need to pass ctx each time
	log := L(ctx)
	log.Info("Processing request")
	log.Warn("Slow query detected")
	log.ErrorWithFields("Request failed", map[string]interface{}{
		"code": 500,
		"path": "/api/users",
	})

	// Output includes trace_id field in all entries
}

func ExampleLogger_levelFiltering() {
	// Set log level to filter messages
	Logger.SetLevel(LevelWarn)

	Logger.Debug("This won't be logged") // filtered out
	Logger.Info("This won't be logged")  // filtered out
	Logger.Warn("This will be logged")   // shown
	Logger.Error("This will be logged")  // shown

	// Set level back to debug
	Logger.SetLevel(LevelDebug)
}

func ExampleSetBaseFields() {
	// Set global fields that will be included in all log entries
	SetBaseFields(map[string]interface{}{
		"service": "my-api",
		"version": "1.0.0",
		"env":     "production",
	})

	// These fields are automatically added to all logs
	Logger.Info("Request received")
	Logger.InfoWithFields("Database query", map[string]interface{}{
		"query": "SELECT * FROM users",
		"rows":  10,
	})

	// Clean up base fields when done
	// SetBaseFields(nil)
}

func ExampleLogger_sync() {
	// Ensure all buffered logs are flushed
	err := Logger.Sync()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
	}
}

func ExampleGetRotatedWriter() {
	// Create a separate log file with rotation
	// Uses the same rotation policy as the main logger
	accessLog := GetRotatedWriter("log/access.log")
	defer accessLog.Close()

	// Write to the separate log file
	accessLog.Write([]byte("GET /api/users 200\n"))
}

func ExampleLogConfig_Validate() {
	// Validate configuration before using it
	cfg := LogConfig{
		Level:  "info",
		Format: "json",
		Outputs: "both",
	}

	err := cfg.Validate()
	if err != nil {
		fmt.Printf("Invalid configuration: %v\n", err)
		return
	}

	// Configuration is valid, use it
	InitLogger(cfg)
}

func ExampleMockLogger() {
	// MockLogger is useful for testing
	mock := NewMockLogger()

	mock.Info("Test message")
	mock.ErrorWithFields("Error occurred", map[string]interface{}{
		"code": 500,
		"err":  "internal error",
	})

	// Check what was logged
	fmt.Println("Last entry:", mock.LastEntry())
	fmt.Println("Has entry:", mock.HasEntry("INFO", "Test message"))

	// Clean up for next test
	mock.Clear()
}
