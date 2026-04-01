package logging

import (
	"errors"
	"testing"
)

func TestFieldString(t *testing.T) {
	f := String("key", "value")
	if f.Key != "key" {
		t.Errorf("expected key 'key', got %q", f.Key)
	}
	if s, ok := f.Value.(string); !ok || s != "value" {
		t.Errorf("expected value 'value', got %v", f.Value)
	}
}

func TestFieldInt(t *testing.T) {
	f := Int("count", 42)
	if f.Key != "count" {
		t.Errorf("expected key 'count', got %q", f.Key)
	}
	if i, ok := f.Value.(int); !ok || i != 42 {
		t.Errorf("expected value 42, got %v", f.Value)
	}
}

func TestFieldInt64(t *testing.T) {
	f := Int64("id", int64(123456789))
	if f.Key != "id" {
		t.Errorf("expected key 'id', got %q", f.Key)
	}
	if i, ok := f.Value.(int64); !ok || i != int64(123456789) {
		t.Errorf("expected value 123456789, got %v", f.Value)
	}
}

func TestFieldFloat64(t *testing.T) {
	f := Float64("ratio", 3.14)
	if f.Key != "ratio" {
		t.Errorf("expected key 'ratio', got %q", f.Key)
	}
	if v, ok := f.Value.(float64); !ok || v != 3.14 {
		t.Errorf("expected value 3.14, got %v", f.Value)
	}
}

func TestFieldBool(t *testing.T) {
	f := Bool("enabled", true)
	if f.Key != "enabled" {
		t.Errorf("expected key 'enabled', got %q", f.Key)
	}
	if b, ok := f.Value.(bool); !ok || b != true {
		t.Errorf("expected value true, got %v", f.Value)
	}
}

func TestFieldErr(t *testing.T) {
	err := errors.New("something failed")
	f := Err(err)
	if f.Key != "error" {
		t.Errorf("expected key 'error', got %q", f.Key)
	}
	if f.Value != err {
		t.Errorf("expected value to be the error, got %v", f.Value)
	}
}

func TestFieldAny(t *testing.T) {
	f := Any("code", 200)
	if f.Key != "code" {
		t.Errorf("expected key 'code', got %q", f.Key)
	}
	if f.Value != 200 {
		t.Errorf("expected value 200, got %v", f.Value)
	}
}

func TestFieldsToMap(t *testing.T) {
	fields := []Field{
		String("name", "test"),
		Int("count", 10),
		Bool("active", true),
	}
	m := FieldsToMap(fields)
	if len(m) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(m))
	}
	if m["name"] != "test" {
		t.Errorf("expected name 'test', got %v", m["name"])
	}
	if m["count"] != 10 {
		t.Errorf("expected count 10, got %v", m["count"])
	}
	if m["active"] != true {
		t.Errorf("expected active true, got %v", m["active"])
	}
}

func TestFieldsToMapEmpty(t *testing.T) {
	m := FieldsToMap(nil)
	if m != nil {
		t.Errorf("expected nil for nil input, got %v", m)
	}

	m = FieldsToMap([]Field{})
	if m != nil {
		t.Errorf("expected nil for empty input, got %v", m)
	}
}

func TestFieldsToMapLaterWins(t *testing.T) {
	fields := []Field{
		String("key", "first"),
		String("key", "second"),
	}
	m := FieldsToMap(fields)
	if m["key"] != "second" {
		t.Errorf("expected later value 'second', got %v", m["key"])
	}
}
