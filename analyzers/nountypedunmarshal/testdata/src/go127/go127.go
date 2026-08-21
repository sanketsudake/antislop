//go:build go1.27

// Package go127 pins the encoding/json/v2 decoders and the jsontext deferred
// decode target, both of which are only importable from Go 1.27. The build tag
// keeps the file out of a Go 1.26 build and sets the file's language version.
package go127

import (
	"bytes"
	"encoding/json/jsontext"
	json "encoding/json/v2"
)

type User struct {
	ID string `json:"id"`
}

// --- invalid: the v2 decoders erase the document exactly like v1 ---

func intoAny(b []byte) error {
	var v any
	return json.Unmarshal(b, &v) // want `nountypedunmarshal: json\.Unmarshal into any keeps the document untyped; decode into a struct that names the fields you use`
}

func intoMap(b []byte) error {
	var m map[string]any
	return json.Unmarshal(b, &m) // want `nountypedunmarshal: json\.Unmarshal into map\[string\]any keeps the document untyped`
}

func readIntoAny(r *bytes.Reader) error {
	var v any
	return json.UnmarshalRead(r, &v) // want `nountypedunmarshal: json\.UnmarshalRead into any keeps the document untyped`
}

func decodeIntoSlice(d *jsontext.Decoder) error {
	var s []any
	return json.UnmarshalDecode(d, &s) // want `nountypedunmarshal: json\.UnmarshalDecode into \[\]any keeps the document untyped`
}

// --- valid ---

func intoStruct(b []byte) error {
	var u User
	return json.Unmarshal(b, &u)
}

// jsontext.Value defers the decode the way json.RawMessage does: its type is
// []byte, not any, so the document is held, not erased.
func intoRaw(b []byte) error {
	var raw jsontext.Value
	return json.Unmarshal(b, &raw)
}

func intoRawMap(b []byte) error {
	var m map[string]jsontext.Value
	return json.Unmarshal(b, &m)
}
