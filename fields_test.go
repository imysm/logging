package logging

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// --- MockLogger *Fields methods ---

func TestMockLogger_AllFieldsMethods(t *testing.T) {
	m := NewMockLogger()

	m.TraceFields("trace msg", String("k", "v"))
	m.DebugFields("debug msg", Int("n", 1))
	m.InfoFields("info msg", Bool("b", true))
	m.WarnFields("warn msg", Float64("f", 1.5))
	m.ErrorFields("error msg", Err(errTest))
	m.FatalFields("fatal msg", String("severity", "fatal"))
	m.PanicFields("panic msg", String("severity", "panic"))

	if len(m.Entries) != 7 {
		t.Fatalf("expected 7 entries, got %d", len(m.Entries))
	}

	expected := []struct {
		level   string
		message string
		key     string
		value   interface{}
	}{
		{"TRACE", "trace msg", "k", "v"},
		{"DEBUG", "debug msg", "n", 1},
		{"INFO", "info msg", "b", true},
		{"WARN", "warn msg", "f", 1.5},
		{"ERROR", "error msg", "error", errTest},
		{"FATAL", "fatal msg", "severity", "fatal"},
		{"PANIC", "panic msg", "severity", "panic"},
	}

	for i, e := range expected {
		entry := m.StructuredEntries[i]
		if entry["level"] != e.level {
			t.Errorf("entry %d: expected level %q, got %v", i, e.level, entry["level"])
		}
		if entry["message"] != e.message {
			t.Errorf("entry %d: expected message %q, got %v", i, e.message, entry["message"])
		}
		if entry[e.key] != e.value {
			t.Errorf("entry %d: expected %s=%v, got %v", i, e.key, e.value, entry[e.key])
		}
	}
}

func TestMockLogger_FieldsMergesBaseFields(t *testing.T) {
	SetBaseFields(map[string]interface{}{"env": "prod"})
	defer SetBaseFields(nil)

	m := NewMockLogger()
	m.InfoFields("with base", String("user", "alice"))

	entry := m.LastStructuredEntry()
	if entry["env"] != "prod" {
		t.Errorf("expected base field env=prod, got %v", entry["env"])
	}
	if entry["user"] != "alice" {
		t.Errorf("expected user=alice, got %v", entry["user"])
	}
}

// --- Real logger *Fields output ---

func TestLogger_FieldsOutput(t *testing.T) {
	tempDir := t.TempDir()
	testLogFile := filepath.Join(tempDir, "fields.log")

	cfg := LogConfig{
		Level:   "trace",
		File:    testLogFile,
		Format:  "json",
		Outputs: "file",
	}
	InitLogger(cfg)

	Logger.InfoFields("structured log", String("user", "bob"), Int("id", 42))

	Logger.Sync()
	time.Sleep(100 * time.Millisecond)

	content, err := readFile(testLogFile)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	if !strings.Contains(content, "structured log") {
		t.Error("log should contain message")
	}
	if !strings.Contains(content, `"user"`) || !strings.Contains(content, `"bob"`) {
		t.Error("log should contain user field")
	}
	if !strings.Contains(content, `"id"`) {
		t.Error("log should contain id field")
	}
}

func TestLogger_AllFieldsOutput(t *testing.T) {
	tempDir := t.TempDir()
	testLogFile := filepath.Join(tempDir, "all_fields.log")

	cfg := LogConfig{
		Level:   "trace",
		File:    testLogFile,
		Format:  "json",
		Outputs: "file",
	}
	InitLogger(cfg)

	Logger.TraceFields("trace f", String("t", "1"))
	Logger.DebugFields("debug f", String("d", "2"))
	Logger.InfoFields("info f", String("i", "3"))
	Logger.WarnFields("warn f", String("w", "4"))
	Logger.ErrorFields("error f", String("e", "5"))

	Logger.Sync()
	time.Sleep(100 * time.Millisecond)

	content, err := readFile(testLogFile)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	for _, msg := range []string{"trace f", "debug f", "info f", "warn f", "error f"} {
		if !strings.Contains(content, msg) {
			t.Errorf("log should contain %q", msg)
		}
	}
}

func TestLogger_FieldsWithCtx(t *testing.T) {
	tempDir := t.TempDir()
	testLogFile := filepath.Join(tempDir, "fields_ctx.log")

	cfg := LogConfig{
		Level:   "debug",
		File:    testLogFile,
		Format:  "json",
		Outputs: "file",
	}
	InitLogger(cfg)

	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-fields-999")

	Logger.InfoFieldsWithCtx(ctx, "ctx fields msg", String("key", "val"))

	Logger.Sync()
	time.Sleep(100 * time.Millisecond)

	content, err := readFile(testLogFile)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	if !strings.Contains(content, "ctx fields msg") {
		t.Error("log should contain message")
	}
}

// --- ContextLogger *Fields methods ---

func TestContextLogger_FieldsMethods(t *testing.T) {
	m := NewMockLogger()
	Logger = m

	ctx := context.Background()
	ctx = WithTraceID(ctx, "ctx-fields-test")
	cl := L(ctx)

	cl.InfoFields("ctx info", String("x", "1"))
	cl.WarnFields("ctx warn", Int("y", 2))
	cl.ErrorFields("ctx error", Bool("z", true))

	if len(m.StructuredEntries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(m.StructuredEntries))
	}

	// All should have trace_id
	for i, e := range m.StructuredEntries {
		if e["trace_id"] != "ctx-fields-test" {
			t.Errorf("entry %d: expected trace_id, got %v", i, e["trace_id"])
		}
	}
	if m.StructuredEntries[0]["x"] != "1" {
		t.Errorf("expected x=1, got %v", m.StructuredEntries[0]["x"])
	}
	if m.StructuredEntries[1]["y"] != 2 {
		t.Errorf("expected y=2, got %v", m.StructuredEntries[1]["y"])
	}
	if m.StructuredEntries[2]["z"] != true {
		t.Errorf("expected z=true, got %v", m.StructuredEntries[2]["z"])
	}
}

func TestContextLogger_TraceDebugFields(t *testing.T) {
	m := NewMockLogger()
	Logger = m

	ctx := context.Background()
	cl := L(ctx)

	cl.TraceFields("trace", String("l", "trace"))
	cl.DebugFields("debug", String("l", "debug"))

	if m.StructuredEntries[0]["level"] != "TRACE" {
		t.Errorf("expected TRACE, got %v", m.StructuredEntries[0]["level"])
	}
	if m.StructuredEntries[1]["level"] != "DEBUG" {
		t.Errorf("expected DEBUG, got %v", m.StructuredEntries[1]["level"])
	}
}

func TestContextLogger_FatalPanicFieldsViaFields(t *testing.T) {
	m := NewMockLogger()
	Logger = m

	ctx := context.Background()
	ctx = WithTraceID(ctx, "fp-fields")
	cl := L(ctx)

	cl.FatalFields("ctx fatal f", String("a", "b"))
	entry := m.LastStructuredEntry()
	if entry["level"] != "FATAL" {
		t.Errorf("expected FATAL, got %v", entry["level"])
	}
	if entry["trace_id"] != "fp-fields" {
		t.Errorf("expected trace_id, got %v", entry["trace_id"])
	}

	m.Clear()
	cl.PanicFields("ctx panic f", Int("c", 3))
	entry = m.LastStructuredEntry()
	if entry["level"] != "PANIC" {
		t.Errorf("expected PANIC, got %v", entry["level"])
	}
}

// --- Validate accepts fatal/panic levels ---

func TestLogConfigValidate_FatalPanic(t *testing.T) {
	for _, lvl := range []string{"fatal", "panic"} {
		t.Run(lvl, func(t *testing.T) {
			cfg := LogConfig{Level: lvl, Format: "text", Outputs: "console"}
			if err := cfg.Validate(); err != nil {
				t.Errorf("expected %q to be a valid log level, got error: %v", lvl, err)
			}
		})
	}
}

// --- Interface compliance ---

func TestContextLogger_StillImplementsInterface(t *testing.T) {
	m := NewMockLogger()
	Logger = m
	cl := L(context.Background())
	var _ LoggerInterface = cl
}
