package encoders

import (
	"encoding/json"
	"io"
)

type Point struct{ X, Y int }

// A container literal handed straight to an encoder is an output boundary:
// the value is serialised on the spot, not carried around untyped.

func marshalLiteral(p Point) ([]byte, error) {
	return json.Marshal(map[string]any{"x": p.X, "y": p.Y})
}

func marshalIndentLiteral(p Point) ([]byte, error) {
	return json.MarshalIndent(map[string]any{"x": p.X}, "", "  ")
}

func encodeLiteral(w io.Writer, p Point) error {
	return json.NewEncoder(w).Encode(map[string]any{"x": p.X, "y": p.Y})
}

func encodePointer(w io.Writer, p Point) error {
	return json.NewEncoder(w).Encode(&map[string]any{"x": p.X})
}

func makeLiteral(w io.Writer) error {
	return json.NewEncoder(w).Encode(make(map[string]any))
}

// The exemption is for the literal in argument position only. A variable
// still carries the untyped map around, and so does a literal passed to a
// function that is not an encoder.

func viaVariable(p Point) ([]byte, error) {
	m := map[string]any{"x": p.X} // want `noanycontainers: map\[string\]any erases the value types`
	return json.Marshal(m)
}

func consume(map[string]any) {} // want `noanycontainers: map\[string\]any erases the value types`

func toOtherFunc(p Point) {
	consume(map[string]any{"x": p.X}) // want `noanycontainers: map\[string\]any erases the value types`
}

// A nested literal inside an encoded literal is part of the same document.
func nested(p Point) ([]byte, error) {
	return json.Marshal(map[string]any{"point": map[string]any{"x": p.X}})
}
