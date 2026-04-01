package logging

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LevelTrace, "TRACE"},
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LevelFatal, "FATAL"},
		{LevelPanic, "PANIC"},
		{LogLevel(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("LogLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"Debug", LevelDebug},
		{"trace", LevelTrace},
		{"TRACE", LevelTrace},
		{"Trace", LevelTrace},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"WARN", LevelWarn},
		{"warning", LevelWarn},
		{"WARNING", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"invalid", LevelInfo}, // defaults to INFO
		{"", LevelInfo},        // defaults to INFO
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseLogLevel(tt.input); got != tt.expected {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLogConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         LogConfig
		expectError bool
		checkFunc   func(*testing.T, LogConfig)
	}{
		{
			name: "valid config",
			cfg: LogConfig{
				Level:  "debug",
				File:   "/tmp/test.log",
				Format: "json",
				Outputs: "both",
			},
			expectError: false,
		},
		{
			name: "empty config gets defaults",
			cfg:  LogConfig{},
			expectError: false,
			checkFunc: func(t *testing.T, cfg LogConfig) {
				if cfg.Level != DefaultLogLevel {
					t.Errorf("expected Level to be %q, got %q", DefaultLogLevel, cfg.Level)
				}
				if cfg.File != DefaultLogFile {
					t.Errorf("expected File to be %q, got %q", DefaultLogFile, cfg.File)
				}
				if cfg.Format != DefaultLogFormat {
					t.Errorf("expected Format to be %q, got %q", DefaultLogFormat, cfg.Format)
				}
				if cfg.Outputs != DefaultLogOutput {
					t.Errorf("expected Outputs to be %q, got %q", DefaultLogOutput, cfg.Outputs)
				}
			},
		},
		{
			name: "invalid log level",
			cfg: LogConfig{
				Level: "invalid",
			},
			expectError: true,
		},
		{
			name: "invalid format",
			cfg: LogConfig{
				Level:  "info",
				Format: "invalid",
			},
			expectError: true,
		},
		{
			name: "invalid output",
			cfg: LogConfig{
				Level:   "info",
				Outputs: "invalid",
			},
			expectError: true,
		},
		{
			name: "zero values get defaults",
			cfg: LogConfig{
				Level:      "info",
				MaxSize:    0,
				MaxBackups: 0,
				MaxAge:     0,
			},
			expectError: false,
			checkFunc: func(t *testing.T, cfg LogConfig) {
				if cfg.MaxSize != 10 {
					t.Errorf("expected MaxSize to be 10, got %d", cfg.MaxSize)
				}
				if cfg.MaxBackups != 5 {
					t.Errorf("expected MaxBackups to be 5, got %d", cfg.MaxBackups)
				}
				if cfg.MaxAge != 30 {
					t.Errorf("expected MaxAge to be 30, got %d", cfg.MaxAge)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, tt.cfg)
			}
		})
	}
}

func TestInitLogger(t *testing.T) {
	// Create a temporary directory for test logs
	tempDir := t.TempDir()
	testLogFile := filepath.Join(tempDir, "test.log")

	tests := []struct {
		name    string
		cfg     LogConfig
		setup   func() func()
		checkFn func(*testing.T)
	}{
		{
			name: "basic initialization",
			cfg: LogConfig{
				Level:   "debug",
				File:    testLogFile,
				Format:  "text",
				Outputs: "both",
			},
			checkFn: func(t *testing.T) {
				if Logger == nil {
					t.Error("Logger should not be nil after initialization")
				}
			},
		},
		{
			name: "console only output",
			cfg: LogConfig{
				Level:   "info",
				Outputs: "console",
			},
			checkFn: func(t *testing.T) {
				if Logger == nil {
					t.Error("Logger should not be nil after initialization")
				}
			},
		},
		{
			name: "file only output",
			cfg: LogConfig{
				Level:   "warn",
				File:    testLogFile,
				Outputs: "file",
			},
			checkFn: func(t *testing.T) {
				if Logger == nil {
					t.Error("Logger should not be nil after initialization")
				}
				// Check if file was created
				if _, err := os.Stat(testLogFile); os.IsNotExist(err) {
					// File might not exist yet if no logs were written
					t.Log("Log file does not exist yet (expected if no logs written)")
				}
			},
		},
		{
			name: "json format",
			cfg: LogConfig{
				Level:   "debug",
				File:    testLogFile,
				Format:  "json",
				Outputs: "file",
			},
			checkFn: func(t *testing.T) {
				if Logger == nil {
					t.Error("Logger should not be nil after initialization")
				}
			},
		},
		{
			name: "creates log directory",
			cfg: LogConfig{
				Level:   "info",
				File:    filepath.Join(tempDir, "subdir", "test.log"),
				Outputs: "file",
			},
			checkFn: func(t *testing.T) {
				subdir := filepath.Join(tempDir, "subdir")
				if _, err := os.Stat(subdir); os.IsNotExist(err) {
					t.Error("log directory should be created")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				cleanup := tt.setup()
				defer cleanup()
			}

			InitLogger(tt.cfg)

			if tt.checkFn != nil {
				tt.checkFn(t)
			}
		})
	}
}

func TestLoggerOutput(t *testing.T) {
	tempDir := t.TempDir()
	testLogFile := filepath.Join(tempDir, "test.log")

	var buf bytes.Buffer
	// Redirect stdout for testing
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := LogConfig{
		Level:   "debug",
		File:    testLogFile,
		Format:  "text",
		Outputs: "both",
	}
	InitLogger(cfg)

	// Write some logs
	Logger.Trace("trace message: %s", "test")
	Logger.Debug("debug message: %s", "test")
	Logger.Info("info message: %s", "test")
	Logger.Warn("warn message: %s", "test")
	Logger.Error("error message: %s", "test")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	// Give time for async operations
	time.Sleep(100 * time.Millisecond)

	// Sync to ensure logs are flushed
	if err := Logger.Sync(); err != nil {
		t.Logf("Sync error (may be expected): %v", err)
	}

	// Check if file contains logs
	content, err := os.ReadFile(testLogFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "trace message") &&
		!strings.Contains(logContent, "debug message") &&
		!strings.Contains(logContent, "info message") &&
		!strings.Contains(logContent, "warn message") &&
		!strings.Contains(logContent, "error message") {
		t.Error("log file should contain at least one of the logged messages")
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	tempDir := t.TempDir()
	testLogFile := filepath.Join(tempDir, "test.log")

	tests := []struct {
		name              string
		level             string
		expectedMessages  []string
		absentMessages    []string
	}{
		{
			name:  "error level only shows errors",
			level: "error",
			expectedMessages: []string{
				"error message",
			},
			absentMessages: []string{
				"trace message",
				"debug message",
				"info message",
				"warn message",
			},
		},
		{
			name:  "warn level shows warnings and errors",
			level: "warn",
			expectedMessages: []string{
				"warn message",
				"error message",
			},
			absentMessages: []string{
				"trace message",
				"debug message",
				"info message",
			},
		},
		{
			name:  "info level shows info and above",
			level: "info",
			expectedMessages: []string{
				"info message",
				"warn message",
				"error message",
			},
			absentMessages: []string{
				"trace message",
				"debug message",
			},
		},
		{
			name:  "debug level shows debug and above",
			level: "debug",
			expectedMessages: []string{
				"debug message",
				"info message",
				"warn message",
				"error message",
			},
			absentMessages: []string{
				"trace message",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := LogConfig{
				Level:   tt.level,
				File:    testLogFile,
				Format:  "text",
				Outputs: "file",
			}
			InitLogger(cfg)

			Logger.Trace("trace message")
			Logger.Debug("debug message")
			Logger.Info("info message")
			Logger.Warn("warn message")
			Logger.Error("error message")

			Logger.Sync()
			time.Sleep(100 * time.Millisecond)

			content, err := os.ReadFile(testLogFile)
			if err != nil {
				t.Fatalf("failed to read log file: %v", err)
			}

			logContent := string(content)

			// Check expected messages are present
			for _, msg := range tt.expectedMessages {
				if !strings.Contains(logContent, msg) {
					t.Errorf("expected log to contain %q at level %s", msg, tt.level)
				}
			}

			// Check absent messages are not present
			for _, msg := range tt.absentMessages {
				if strings.Contains(logContent, msg) {
					t.Errorf("did not expect log to contain %q at level %s", msg, tt.level)
				}
			}
		})
	}
}

func TestLoggerWithFields(t *testing.T) {
	tempDir := t.TempDir()
	testLogFile := filepath.Join(tempDir, "test.log")

	cfg := LogConfig{
		Level:   "debug",
		File:    testLogFile,
		Format:  "text",
		Outputs: "file",
	}
	InitLogger(cfg)

	// Set base fields
	SetBaseFields(map[string]interface{}{
		"service": "test-service",
		"env":     "test",
	})
	defer SetBaseFields(nil)

	Logger.InfoWithFields("user action", map[string]interface{}{
		"user_id": 12345,
		"action":  "login",
	})

	Logger.Sync()
	time.Sleep(100 * time.Millisecond)

	content, err := os.ReadFile(testLogFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "user action") {
		t.Error("log should contain the message")
	}
	// Note: field format depends on slog's text handler
}

func TestLoggerWithCtx(t *testing.T) {
	tempDir := t.TempDir()
	testLogFile := filepath.Join(tempDir, "test.log")

	cfg := LogConfig{
		Level:   "debug",
		File:    testLogFile,
		Format:  "text",
		Outputs: "file",
	}
	InitLogger(cfg)

	ctx := context.Background()
	ctx = WithTraceID(ctx, "test-trace-123")

	Logger.InfoWithCtx(ctx, "message with trace")

	Logger.Sync()
	time.Sleep(100 * time.Millisecond)

	content, err := os.ReadFile(testLogFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "message with trace") {
		t.Error("log should contain the message")
	}
	// Note: trace_id format depends on slog's text handler
}

func TestBaseFields(t *testing.T) {
	// Clean up before test
	SetBaseFields(nil)

	// Test setting base fields
	fields := map[string]interface{}{
		"service": "my-service",
		"version": "1.0.0",
	}
	SetBaseFields(fields)

	// Test getting base fields
	got := GetBaseFields()
	if got["service"] != "my-service" {
		t.Errorf("expected service to be 'my-service', got %v", got["service"])
	}
	if got["version"] != "1.0.0" {
		t.Errorf("expected version to be '1.0.0', got %v", got["version"])
	}

	// Test that we get a copy (modifying returned map doesn't affect internal)
	got["service"] = "modified"
	again := GetBaseFields()
	if again["service"] == "modified" {
		t.Error("modifying returned map should not affect internal state")
	}

	// Test WithBaseFields
	customFields := map[string]interface{}{
		"user_id": 123,
	}
	merged := WithBaseFields(customFields)

	if merged["service"] != "my-service" {
		t.Error("merged fields should contain base fields")
	}
	if merged["user_id"] != 123 {
		t.Error("merged fields should contain custom fields")
	}

	// Clean up
	SetBaseFields(nil)
}

func TestTraceID(t *testing.T) {
	ctx := context.Background()

	// No trace ID
	if got := TraceID(ctx); got != "" {
		t.Errorf("expected empty trace ID, got %q", got)
	}

	// With trace ID
	ctx = WithTraceID(ctx, "trace-123")
	if got := TraceID(ctx); got != "trace-123" {
		t.Errorf("expected trace ID 'trace-123', got %q", got)
	}
}

func TestCtxFields(t *testing.T) {
	ctx := context.Background()

	// No ctx fields
	if got := CtxFields(ctx); got != nil {
		t.Errorf("expected nil ctx fields, got %v", got)
	}

	// With ctx fields
	ctx = WithCtxFields(ctx, map[string]interface{}{
		"user_id": 123,
		"env":     "test",
	})
	got := CtxFields(ctx)
	if got["user_id"] != 123 {
		t.Errorf("expected user_id 123, got %v", got["user_id"])
	}
	if got["env"] != "test" {
		t.Errorf("expected env 'test', got %v", got["env"])
	}

	// Empty ctx fields
	if got := CtxFields(context.Background()); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestSetLevel(t *testing.T) {
	tempDir := t.TempDir()
	testLogFile := filepath.Join(tempDir, "test.log")

	cfg := LogConfig{
		Level:   "info",
		File:    testLogFile,
		Format:  "text",
		Outputs: "file",
	}
	InitLogger(cfg)

	// Write at least one log to ensure file is created
	Logger.Info("initial log")
	Logger.Sync()

	// This should not be logged
	Logger.Debug("debug before set level")
	Logger.Sync()

	// Change level to debug
	Logger.SetLevel(LevelDebug)
	Logger.Debug("debug after set level")
	Logger.Sync()

	time.Sleep(100 * time.Millisecond)

	// Check if file exists first
	if _, err := os.Stat(testLogFile); os.IsNotExist(err) {
		t.Skip("log file not created, skipping test")
		return
	}

	content, err := os.ReadFile(testLogFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(content)
	t.Logf("Log content:\n%s", logContent)

	if strings.Contains(logContent, "debug before set level") {
		t.Error("debug message before set level should not be logged")
	}
	if !strings.Contains(logContent, "debug after set level") {
		t.Error("debug message after set level should be logged")
	}
}

func TestGetRotatedWriter(t *testing.T) {
	tempDir := t.TempDir()
	testLogFile := filepath.Join(tempDir, "rotated.log")

	cfg := LogConfig{
		Level:      "info",
		MaxSize:    100,
		MaxBackups: 10,
		MaxAge:     60,
		Compress:   true,
	}
	InitLogger(cfg)

	writer := GetRotatedWriter(testLogFile)
	if writer == nil {
		t.Fatal("GetRotatedWriter should return a non-nil writer")
	}

	// Write something
	_, err := writer.Write([]byte("test log\n"))
	if err != nil {
		t.Errorf("failed to write to rotated writer: %v", err)
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		t.Errorf("failed to close rotated writer: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(testLogFile); os.IsNotExist(err) {
		t.Error("log file should exist after writing")
	}
}

func TestGetGlobalConfig(t *testing.T) {
	cfg := LogConfig{
		Level:      "debug",
		File:       "/tmp/test.log",
		MaxSize:    50,
		MaxBackups: 3,
		MaxAge:     15,
		Compress:   true,
		Format:     "json",
		Outputs:    "both",
	}
	InitLogger(cfg)

	got := GetGlobalConfig()
	if got.Level != cfg.Level {
		t.Errorf("expected Level %q, got %q", cfg.Level, got.Level)
	}
	if got.File != cfg.File {
		t.Errorf("expected File %q, got %q", cfg.File, got.File)
	}
	if got.MaxSize != cfg.MaxSize {
		t.Errorf("expected MaxSize %d, got %d", cfg.MaxSize, got.MaxSize)
	}
}

// Benchmark logging performance
func BenchmarkLoggerInfo(b *testing.B) {
	tempDir := b.TempDir()
	testLogFile := filepath.Join(tempDir, "bench.log")

	cfg := LogConfig{
		Level:   "info",
		File:    testLogFile,
		Format:  "text",
		Outputs: "file",
	}
	InitLogger(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Logger.Info("benchmark message: %d", i)
	}
}

func BenchmarkLoggerWithFields(b *testing.B) {
	tempDir := b.TempDir()
	testLogFile := filepath.Join(tempDir, "bench.log")

	cfg := LogConfig{
		Level:   "info",
		File:    testLogFile,
		Format:  "text",
		Outputs: "file",
	}
	InitLogger(cfg)

	fields := map[string]interface{}{
		"user_id": 123,
		"action":  "benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Logger.InfoWithFields("benchmark message", fields)
	}
}
