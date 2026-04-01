package logging

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockLogger is a structured logging implementation for unit tests.
// It preserves both raw text and structured entries for verification.
type MockLogger struct {
	Entries           []string
	StructuredEntries []map[string]interface{}
	mu                sync.Mutex
}

// NewMockLogger creates a new MockLogger instance.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		Entries:           make([]string, 0),
		StructuredEntries: make([]map[string]interface{}, 0),
	}
}

// add is a helper method to add log entries. Returns the index of the created entry.
func (m *MockLogger) add(level, format string, v ...interface{}) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg := fmt.Sprintf(format, v...)
	m.Entries = append(m.Entries, fmt.Sprintf("[%s] %s", level, msg))
	entry := map[string]interface{}{
		"level":     level,
		"timestamp": time.Now().Format(time.RFC3339),
		"message":   msg,
	}
	m.StructuredEntries = append(m.StructuredEntries, entry)
	return len(m.StructuredEntries) - 1
}

// addWithFields is a helper method for structured logging with fields
func (m *MockLogger) addWithFields(level, format string, fields map[string]interface{}, v ...interface{}) {
	idx := m.add(level, format, v...)
	m.mu.Lock()
	defer m.mu.Unlock()

	e := m.StructuredEntries[idx]

	if base := GetBaseFields(); base != nil {
		for k, vv := range base {
			e[k] = vv
		}
	}

	for k, vv := range fields {
		e[k] = vv
	}
}

// addWithCtx is a helper method for context-aware logging
func (m *MockLogger) addWithCtx(ctx context.Context, level, format string, v ...interface{}) {
	idx := m.add(level, format, v...)
	m.mu.Lock()
	defer m.mu.Unlock()

	e := m.StructuredEntries[idx]

	if traceID := TraceID(ctx); traceID != "" {
		e["trace_id"] = traceID
	}

	// Merge ctx fields
	for k, v := range CtxFields(ctx) {
		e[k] = v
	}

	// Merge base fields
	if base := GetBaseFields(); base != nil {
		for k, vv := range base {
			e[k] = vv
		}
	}
}

func (m *MockLogger) Trace(format string, v ...interface{}) {
	m.add("TRACE", format, v...)
}

func (m *MockLogger) Debug(format string, v ...interface{}) {
	m.add("DEBUG", format, v...)
}

func (m *MockLogger) Info(format string, v ...interface{}) {
	m.add("INFO", format, v...)
}

func (m *MockLogger) Warn(format string, v ...interface{}) {
	m.add("WARN", format, v...)
}

func (m *MockLogger) Error(format string, v ...interface{}) {
	m.add("ERROR", format, v...)
}

func (m *MockLogger) SetLevel(level LogLevel) {
	// MockLogger doesn't filter by level
}

// Sync implements LoggerInterface
func (m *MockLogger) Sync() error {
	return nil
}

// WithFields variants for structured logging used in tests
func (m *MockLogger) TraceWithFields(format string, fields map[string]interface{}, v ...interface{}) {
	m.addWithFields("TRACE", format, fields, v...)
}

func (m *MockLogger) DebugWithFields(format string, fields map[string]interface{}, v ...interface{}) {
	m.addWithFields("DEBUG", format, fields, v...)
}

func (m *MockLogger) InfoWithFields(format string, fields map[string]interface{}, v ...interface{}) {
	m.addWithFields("INFO", format, fields, v...)
}

func (m *MockLogger) WarnWithFields(format string, fields map[string]interface{}, v ...interface{}) {
	m.addWithFields("WARN", format, fields, v...)
}

func (m *MockLogger) ErrorWithFields(format string, fields map[string]interface{}, v ...interface{}) {
	m.addWithFields("ERROR", format, fields, v...)
}

// Context-aware logging methods
func (m *MockLogger) TraceWithCtx(ctx context.Context, format string, v ...interface{}) {
	m.addWithCtx(ctx, "TRACE", format, v...)
}

func (m *MockLogger) DebugWithCtx(ctx context.Context, format string, v ...interface{}) {
	m.addWithCtx(ctx, "DEBUG", format, v...)
}

func (m *MockLogger) InfoWithCtx(ctx context.Context, format string, v ...interface{}) {
	m.addWithCtx(ctx, "INFO", format, v...)
}

func (m *MockLogger) WarnWithCtx(ctx context.Context, format string, v ...interface{}) {
	m.addWithCtx(ctx, "WARN", format, v...)
}

func (m *MockLogger) ErrorWithCtx(ctx context.Context, format string, v ...interface{}) {
	m.addWithCtx(ctx, "ERROR", format, v...)
}

// Clear clears all logged entries. Useful for test isolation.
func (m *MockLogger) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Entries = make([]string, 0)
	m.StructuredEntries = make([]map[string]interface{}, 0)
}

// HasEntry checks if an entry with the given level and message exists.
func (m *MockLogger) HasEntry(level, message string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.Entries {
		if fmt.Sprintf("[%s] %s", level, message) == e {
			return true
		}
	}
	return false
}

// LastEntry returns the last log entry as a string.
func (m *MockLogger) LastEntry() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Entries) == 0 {
		return ""
	}
	return m.Entries[len(m.Entries)-1]
}

// LastStructuredEntry returns the last structured log entry.
func (m *MockLogger) LastStructuredEntry() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.StructuredEntries) == 0 {
		return nil
	}
	return m.StructuredEntries[len(m.StructuredEntries)-1]
}
