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
