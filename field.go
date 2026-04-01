package logging

// Field is a type-safe key-value pair for structured logging.
// Use the constructor functions (String, Int, Err, etc.) to create instances.
type Field struct {
	Key   string
	Value interface{}
}

// String creates a Field with a string value.
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates a Field with an int value.
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates a Field with an int64 value.
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a Field with a float64 value.
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a Field with a bool value.
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Err creates a Field with key "error" and the given error value.
func Err(err error) Field {
	return Field{Key: "error", Value: err}
}

// Any creates a Field with an arbitrary value.
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// FieldsToMap converts a slice of Fields to a map[string]interface{}.
// Returns nil if the input is empty. Later fields overwrite earlier ones with the same key.
func FieldsToMap(fields []Field) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}
	m := make(map[string]interface{}, len(fields))
	for _, f := range fields {
		m[f.Key] = f.Value
	}
	return m
}
