package logging

import (
	"context"
	"testing"
)

func TestMockLogger_MergesBaseFields(t *testing.T) {
	m := NewMockLogger()
	// set base fields
	SetBaseFields(map[string]interface{}{"host": "h1", "env": "dev"})
	defer SetBaseFields(nil)

	// call InfoWithFields
	m.InfoWithFields("hello %s", map[string]interface{}{"k": "v"}, "world")

	if len(m.StructuredEntries) == 0 {
		t.Fatalf("expected structured entries, got none")
	}
	e := m.StructuredEntries[len(m.StructuredEntries)-1]
	if e["host"] != "h1" {
		t.Fatalf("expected host base field present, got %#v", e)
	}
	if e["k"] != "v" {
		t.Fatalf("expected k field present, got %#v", e)
	}
	if e["message"] == "" {
		t.Fatalf("expected message field present, got %#v", e)
	}
}

func TestMockLogger_HasEntry(t *testing.T) {
	m := NewMockLogger()

	m.Info("test message")

	if !m.HasEntry("INFO", "test message") {
		t.Error("expected to find the entry")
	}

	if m.HasEntry("DEBUG", "test message") {
		t.Error("should not find entry with different level")
	}
}

func TestMockLogger_LastEntry(t *testing.T) {
	m := NewMockLogger()

	m.Info("first message")
	m.Info("second message")

	last := m.LastEntry()
	if last != "[INFO] second message" {
		t.Errorf("expected last entry to be '[INFO] second message', got %q", last)
	}
}

func TestMockLogger_LastStructuredEntry(t *testing.T) {
	m := NewMockLogger()

	m.Info("test message")

	entry := m.LastStructuredEntry()
	if entry == nil {
		t.Fatal("expected structured entry, got nil")
	}

	if entry["level"] != "INFO" {
		t.Errorf("expected level INFO, got %v", entry["level"])
	}
	if entry["message"] != "test message" {
		t.Errorf("expected message 'test message', got %v", entry["message"])
	}
	if entry["timestamp"] == "" {
		t.Error("expected timestamp to be set")
	}
}

func TestMockLogger_Clear(t *testing.T) {
	m := NewMockLogger()

	m.Info("message 1")
	m.Info("message 2")

	if len(m.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(m.Entries))
	}

	m.Clear()

	if len(m.Entries) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(m.Entries))
	}
	if len(m.StructuredEntries) != 0 {
		t.Errorf("expected 0 structured entries after clear, got %d", len(m.StructuredEntries))
	}
}

func TestMockLogger_WithCtx(t *testing.T) {
	m := NewMockLogger()

	ctx := context.Background()
	ctx = WithTraceID(ctx, "test-trace-123")

	m.InfoWithCtx(ctx, "test message")

	entry := m.LastStructuredEntry()
	if entry == nil {
		t.Fatal("expected structured entry, got nil")
	}

	if entry["trace_id"] != "test-trace-123" {
		t.Errorf("expected trace_id 'test-trace-123', got %v", entry["trace_id"])
	}
}

func TestMockLogger_AllLogLevels(t *testing.T) {
	m := NewMockLogger()

	m.Trace("trace message")
	m.Debug("debug message")
	m.Info("info message")
	m.Warn("warn message")
	m.Error("error message")

	if len(m.Entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(m.Entries))
	}

	expected := []string{
		"[TRACE] trace message",
		"[DEBUG] debug message",
		"[INFO] info message",
		"[WARN] warn message",
		"[ERROR] error message",
	}

	for i, exp := range expected {
		if m.Entries[i] != exp {
			t.Errorf("entry %d: expected %q, got %q", i, exp, m.Entries[i])
		}
	}
}

func TestMockLogger_WithFields(t *testing.T) {
	m := NewMockLogger()

	m.InfoWithFields("test message", map[string]interface{}{
		"user_id": 123,
		"action":  "test",
	})

	entry := m.LastStructuredEntry()
	if entry == nil {
		t.Fatal("expected structured entry, got nil")
	}

	if entry["user_id"] != 123 {
		t.Errorf("expected user_id 123, got %v", entry["user_id"])
	}
	if entry["action"] != "test" {
		t.Errorf("expected action 'test', got %v", entry["action"])
	}
}

func TestMockLogger_ConcurrentWrites(t *testing.T) {
	m := NewMockLogger()
	done := make(chan bool)

	// Write from multiple goroutines
	for i := 0; i < 10; i++ {
		go func(id int) {
			m.Info("message from goroutine %d", id)
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}

	if len(m.Entries) != 10 {
		t.Errorf("expected 10 entries, got %d", len(m.Entries))
	}
	if len(m.StructuredEntries) != 10 {
		t.Errorf("expected 10 structured entries, got %d", len(m.StructuredEntries))
	}
}

func TestContextLogger_Basic(t *testing.T) {
	m := NewMockLogger()
	Logger = m

	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-123")

	cl := L(ctx)
	cl.Trace("trace msg")
	cl.Debug("debug msg")
	cl.Info("info msg")
	cl.Warn("warn msg")
	cl.Error("error msg")

	if len(m.Entries) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(m.Entries))
	}

	// Check trace_id is present in all entries
	for i, e := range m.StructuredEntries {
		if e["trace_id"] != "trace-123" {
			t.Errorf("entry %d: expected trace_id 'trace-123', got %v", i, e["trace_id"])
		}
	}
}

func TestContextLogger_WithFields(t *testing.T) {
	m := NewMockLogger()
	Logger = m

	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-456")

	cl := L(ctx)
	cl.InfoWithFields("user login", map[string]interface{}{
		"user_id": 123,
	})

	entry := m.LastStructuredEntry()
	if entry == nil {
		t.Fatal("expected structured entry, got nil")
	}

	if entry["trace_id"] != "trace-456" {
		t.Errorf("expected trace_id 'trace-456', got %v", entry["trace_id"])
	}
	if entry["user_id"] != 123 {
		t.Errorf("expected user_id 123, got %v", entry["user_id"])
	}
}

func TestContextLogger_ImplementsInterface(t *testing.T) {
	m := NewMockLogger()
	Logger = m

	ctx := context.Background()
	cl := L(ctx)

	// Verify ContextLogger satisfies LoggerInterface
	var _ LoggerInterface = cl

	// Basic smoke test for SetLevel and Sync
	cl.SetLevel(LevelDebug)
	if err := cl.Sync(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContextLogger_WithoutTraceID(t *testing.T) {
	m := NewMockLogger()
	Logger = m

	ctx := context.Background()
	cl := L(ctx)
	cl.Info("no trace id")

	entry := m.LastStructuredEntry()
	if entry == nil {
		t.Fatal("expected structured entry, got nil")
	}

	if _, ok := entry["trace_id"]; ok {
		t.Error("trace_id should not be present when not set in context")
	}
}

func TestContextLogger_WithCtxFields(t *testing.T) {
	m := NewMockLogger()
	Logger = m

	ctx := context.Background()
	ctx = WithCtxFields(ctx, map[string]interface{}{
		"user_id":    123,
		"request_id": "req-abc",
	})

	cl := L(ctx)
	cl.Info("processing request")
	cl.InfoWithFields("user login", map[string]interface{}{
		"ip": "1.2.3.4",
	})

	if len(m.StructuredEntries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m.StructuredEntries))
	}

	// First entry: basic log with ctx fields
	e1 := m.StructuredEntries[0]
	if e1["user_id"] != 123 {
		t.Errorf("expected user_id 123, got %v", e1["user_id"])
	}
	if e1["request_id"] != "req-abc" {
		t.Errorf("expected request_id 'req-abc', got %v", e1["request_id"])
	}

	// Second entry: WithFields merged with ctx fields
	e2 := m.StructuredEntries[1]
	if e2["user_id"] != 123 {
		t.Errorf("expected user_id 123, got %v", e2["user_id"])
	}
	if e2["ip"] != "1.2.3.4" {
		t.Errorf("expected ip '1.2.3.4', got %v", e2["ip"])
	}
}

func TestContextLogger_WithCtxFieldsMerge(t *testing.T) {
	m := NewMockLogger()
	Logger = m

	ctx := context.Background()
	ctx = WithCtxFields(ctx, map[string]interface{}{
		"trace_id": "override-this",
		"user_id":  1,
	})
	// Second call merges, does not replace
	ctx = WithCtxFields(ctx, map[string]interface{}{
		"request_id": "req-xyz",
		"user_id":    2, // override previous
	})

	fields := CtxFields(ctx)
	if fields["user_id"] != 2 {
		t.Errorf("expected user_id 2 (overridden), got %v", fields["user_id"])
	}
	if fields["request_id"] != "req-xyz" {
		t.Errorf("expected request_id 'req-xyz', got %v", fields["request_id"])
	}
	if fields["trace_id"] != "override-this" {
		t.Errorf("expected trace_id preserved, got %v", fields["trace_id"])
	}
}
