package noencoders

import "encoding/json"

// With encoders set to the empty list the literal is reported like any other.
func marshalLiteral() ([]byte, error) {
	return json.Marshal(map[string]any{"x": 1}) // want `noanycontainers: map\[string\]any erases the value types`
}
