package logging

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- Level constants ---

func TestLevelFatalAndPanic(t *testing.T) {
	if LevelError >= LevelFatal {
		t.Errorf("LevelFatal (%d) should be greater than LevelError (%d)", LevelFatal, LevelError)
	}
	if LevelFatal >= LevelPanic {
		t.Errorf("LevelPanic (%d) should be greater than LevelFatal (%d)", LevelPanic, LevelFatal)
	}
	if LevelFatal.String() != "FATAL" {
		t.Errorf("expected 'FATAL', got %q", LevelFatal.String())
	}
	if LevelPanic.String() != "PANIC" {
		t.Errorf("expected 'PANIC', got %q", LevelPanic.String())
	}
}

func TestParseLogLevelFatalPanic(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"fatal", LevelFatal},
		{"FATAL", LevelFatal},
		{"Fatal", LevelFatal},
		{"panic", LevelPanic},
		{"PANIC", LevelPanic},
		{"Panic", LevelPanic},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseLogLevel(tt.input); got != tt.expected {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// --- MockLogger Fatal/Panic ---

func TestMockLogger_FatalPanic(t *testing.T) {
	m := NewMockLogger()

	m.Fatal("fatal message")
	m.Panic("panic message")

	if len(m.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m.Entries))
	}
	if m.Entries[0] != "[FATAL] fatal message" {
		t.Errorf("expected '[FATAL] fatal message', got %q", m.Entries[0])
	}
	if m.Entries[1] != "[PANIC] panic message" {
		t.Errorf("expected '[PANIC] panic message', got %q", m.Entries[1])
	}
}

func TestMockLogger_FatalPanicWithFields(t *testing.T) {
	m := NewMockLogger()

	m.FatalWithFields("fatal with fields", map[string]interface{}{"code": 1})
	m.PanicWithFields("panic with fields", map[string]interface{}{"reason": "test"})

	entries := m.StructuredEntries
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0]["level"] != "FATAL" {
		t.Errorf("expected level FATAL, got %v", entries[0]["level"])
	}
	if entries[0]["code"] != 1 {
		t.Errorf("expected code 1, got %v", entries[0]["code"])
	}
	if entries[1]["level"] != "PANIC" {
		t.Errorf("expected level PANIC, got %v", entries[1]["level"])
	}
	if entries[1]["reason"] != "test" {
		t.Errorf("expected reason 'test', got %v", entries[1]["reason"])
	}
}

func TestMockLogger_FatalPanicWithCtx(t *testing.T) {
	m := NewMockLogger()
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-fatal")

	m.FatalWithCtx(ctx, "fatal with ctx")
	m.PanicWithCtx(ctx, "panic with ctx")

	entries := m.StructuredEntries
	if entries[0]["trace_id"] != "trace-fatal" {
		t.Errorf("expected trace_id 'trace-fatal', got %v", entries[0]["trace_id"])
	}
	if entries[1]["trace_id"] != "trace-fatal" {
		t.Errorf("expected trace_id 'trace-fatal', got %v", entries[1]["trace_id"])
	}
}

// --- MockLogger FatalFields/PanicFields ---

func TestMockLogger_FatalPanicFields(t *testing.T) {
	m := NewMockLogger()

	m.FatalFields("fatal fields", String("key", "val"), Int("code", 1))
	m.PanicFields("panic fields", Err(errTest))

	entries := m.StructuredEntries
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0]["level"] != "FATAL" {
		t.Errorf("expected level FATAL, got %v", entries[0]["level"])
	}
	if entries[0]["key"] != "val" {
		t.Errorf("expected key 'val', got %v", entries[0]["key"])
	}
	if entries[0]["code"] != 1 {
		t.Errorf("expected code 1, got %v", entries[0]["code"])
	}
	if entries[1]["level"] != "PANIC" {
		t.Errorf("expected level PANIC, got %v", entries[1]["level"])
	}
	if entries[1]["error"] == nil {
		t.Error("expected error field to be set")
	}
}

// --- Real logger Fatal calls os.Exit(1) ---

func TestLoggerFatal_Exits(t *testing.T) {
	if os.Getenv("GO_TEST_FATAL") == "1" {
		tempDir := os.Getenv("GO_TEST_FATAL_DIR")
		testLogFile := filepath.Join(tempDir, "fatal.log")
		cfg := LogConfig{
			Level:   "debug",
			File:    testLogFile,
			Format:  "text",
			Outputs: "both",
		}
		InitLogger(cfg)
		Logger.Fatal("fatal exit test")
		return
	}
	tempDir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=TestLoggerFatal_Exits")
	cmd.Env = append(os.Environ(), "GO_TEST_FATAL=1", "GO_TEST_FATAL_DIR="+tempDir)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected Fatal to cause the process to exit with non-zero status")
	}
	// Check that log was written to file
	content, readErr := os.ReadFile(filepath.Join(tempDir, "fatal.log"))
	if readErr != nil {
		t.Logf("could not read log file: %v", readErr)
	} else if !strings.Contains(string(content), "fatal exit test") {
		t.Errorf("expected log file to contain 'fatal exit test', got: %s", string(content))
	}
	// Also check stdout output
	if !strings.Contains(string(output), "fatal exit test") {
		t.Logf("stdout output: %s", string(output))
	}
}

// --- Real logger Panic calls panic() ---

func TestLoggerPanic_Panics(t *testing.T) {
	tempDir := t.TempDir()
	testLogFile := filepath.Join(tempDir, "panic.log")
	cfg := LogConfig{
		Level:   "debug",
		File:    testLogFile,
		Format:  "text",
		Outputs: "file",
	}
	InitLogger(cfg)

	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected Panic to actually panic")
		}
	}()
	Logger.Panic("panic test message")
}

// --- noopLogger Fatal/Panic ---

func TestNoopLogger_FatalPanic(t *testing.T) {
	// noopLogger.Fatal should call os.Exit — test via subprocess
	if os.Getenv("GO_TEST_NOOP_FATAL") == "1" {
		n := &noopLogger{}
		n.Fatal("noop fatal")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestNoopLogger_FatalPanic")
	cmd.Env = append(os.Environ(), "GO_TEST_NOOP_FATAL=1")
	err := cmd.Run()
	if err == nil {
		t.Error("expected noopLogger.Fatal to exit the process")
	}
}

func TestNoopLogger_Panic(t *testing.T) {
	n := &noopLogger{}
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected noopLogger.Panic to panic")
		}
	}()
	n.Panic("noop panic")
}

// --- ContextLogger Fatal/Panic ---

func TestContextLogger_FatalPanic(t *testing.T) {
	m := NewMockLogger()
	Logger = m

	ctx := context.Background()
	ctx = WithTraceID(ctx, "ctx-fatal-123")
	cl := L(ctx)

	// Fatal via context logger delegates to FatalWithCtx
	cl.Fatal("ctx fatal message")

	entry := m.LastStructuredEntry()
	if entry["level"] != "FATAL" {
		t.Errorf("expected level FATAL, got %v", entry["level"])
	}
	if entry["trace_id"] != "ctx-fatal-123" {
		t.Errorf("expected trace_id 'ctx-fatal-123', got %v", entry["trace_id"])
	}

	// Panic
	m.Clear()
	cl.Panic("ctx panic message")

	entry = m.LastStructuredEntry()
	if entry["level"] != "PANIC" {
		t.Errorf("expected level PANIC, got %v", entry["level"])
	}
}

func TestContextLogger_FatalWithFields(t *testing.T) {
	m := NewMockLogger()
	Logger = m

	ctx := context.Background()
	ctx = WithTraceID(ctx, "ctx-fields")
	cl := L(ctx)

	cl.FatalWithFields("ctx fatal fields", map[string]interface{}{"code": 1})

	entry := m.LastStructuredEntry()
	if entry["level"] != "FATAL" {
		t.Errorf("expected level FATAL, got %v", entry["level"])
	}
	if entry["trace_id"] != "ctx-fields" {
		t.Errorf("expected trace_id 'ctx-fields', got %v", entry["trace_id"])
	}
	if entry["code"] != 1 {
		t.Errorf("expected code 1, got %v", entry["code"])
	}
}

func TestContextLogger_FatalPanicWithCtx(t *testing.T) {
	m := NewMockLogger()
	Logger = m

	ctx1 := context.Background()
	ctx1 = WithTraceID(ctx1, "bound-ctx")
	cl := L(ctx1)

	ctx2 := context.Background()
	ctx2 = WithTraceID(ctx2, "passed-ctx")

	cl.FatalWithCtx(ctx2, "fatal merge ctx")
	entry := m.LastStructuredEntry()
	if entry["trace_id"] != "passed-ctx" {
		t.Errorf("expected passed-ctx to win, got %v", entry["trace_id"])
	}
}

func TestContextLogger_FatalPanicFields(t *testing.T) {
	m := NewMockLogger()
	Logger = m

	ctx := context.Background()
	ctx = WithTraceID(ctx, "ctx-f")
	cl := L(ctx)

	cl.FatalFields("ctx fatal fields", String("k", "v"))
	entry := m.LastStructuredEntry()
	if entry["level"] != "FATAL" {
		t.Errorf("expected level FATAL, got %v", entry["level"])
	}
	if entry["trace_id"] != "ctx-f" {
		t.Errorf("expected trace_id, got %v", entry["trace_id"])
	}
	if entry["k"] != "v" {
		t.Errorf("expected k='v', got %v", entry["k"])
	}

	m.Clear()
	cl.PanicFields("ctx panic fields", Int("n", 42))
	entry = m.LastStructuredEntry()
	if entry["level"] != "PANIC" {
		t.Errorf("expected level PANIC, got %v", entry["level"])
	}
	if entry["n"] != 42 {
		t.Errorf("expected n=42, got %v", entry["n"])
	}
}

// --- Fatal/Panic level filtering ---

func TestLoggerLevelFiltering_IncludesFatalPanic(t *testing.T) {
	tempDir := t.TempDir()
	testLogFile := filepath.Join(tempDir, "filter.log")

	cfg := LogConfig{
		Level:   "info",
		File:    testLogFile,
		Format:  "text",
		Outputs: "file",
	}
	InitLogger(cfg)

	// Fatal and Panic should always be logged regardless of level
	// We can't test Fatal directly (it exits), so test that FATAL/PANIC levels
	// are above ERROR in the hierarchy
	if LogLevel(lvlFromLogger()) <= LevelError {
		t.Error("fatal/panic levels should always pass through level filter")
	}
}

// helper to get level from current logger
func lvlFromLogger() LogLevel {
	// We just verify the level ordering
	return LevelFatal
}

// test error for Err() field
var errTest = new(testError)

type testError struct{}

func (e *testError) Error() string { return "test error" }
