package a

import (
	"bytes"
	"encoding/json"
	"text/template"

	"gopkg.in/yaml.v3"
)

type User struct {
	ID string `json:"id"`
}

// Doc is declared here, so its shape is this package's decision.
type Doc map[string]any

// --- invalid ---

func intoAny(b []byte) error {
	var v any
	return json.Unmarshal(b, &v) // want `nountypedunmarshal: json\.Unmarshal into any keeps the document untyped; decode into a struct that names the fields you use`
}

func intoMap(b []byte) error {
	var m map[string]any
	return json.Unmarshal(b, &m) // want `nountypedunmarshal: json\.Unmarshal into map\[string\]any keeps the document untyped`
}

func intoSlice(b []byte) error {
	var l []any
	return json.Unmarshal(b, &l) // want `nountypedunmarshal: json\.Unmarshal into \[\]any keeps the document untyped`
}

func intoNamedMap(b []byte) error {
	var d Doc
	return json.Unmarshal(b, &d) // want `nountypedunmarshal: json\.Unmarshal into map\[string\]any keeps the document untyped`
}

func intoDecoder(r *bytes.Reader) error {
	var v any
	return json.NewDecoder(r).Decode(&v) // want `nountypedunmarshal: json\.Decoder\.Decode into any keeps the document untyped`
}

func intoYAML(b []byte) error {
	var m map[string]any
	return yaml.Unmarshal(b, &m) // want `nountypedunmarshal: yaml\.Unmarshal into map\[string\]any keeps the document untyped`
}

func intoYAMLDecoder(r *bytes.Reader) error {
	var v any
	return yaml.NewDecoder(r).Decode(&v) // want `nountypedunmarshal: yaml\.Decoder\.Decode into any keeps the document untyped`
}

// --- valid ---

// The struct names the fields this code reads.
func intoStruct(b []byte) error {
	var u User
	return json.Unmarshal(b, &u)
}

// json.RawMessage defers the decode; its type is []byte, not any.
func intoRaw(b []byte) error {
	var raw json.RawMessage
	return json.Unmarshal(b, &raw)
}

// The values keep a type that says "not decoded yet".
func intoRawMap(b []byte) error {
	var m map[string]json.RawMessage
	return json.Unmarshal(b, &m)
}

// An imported named type is that package's contract, not this package's.
func intoFuncMap(b []byte) error {
	var fns template.FuncMap
	return json.Unmarshal(b, &fns)
}

// Marshal takes a value to encode, not a decode target.
func marshal(u User) ([]byte, error) { return json.Marshal(u) }

// A function that is not in the list decodes nothing as far as this rule knows.
func decodeElsewhere(b []byte, into func([]byte, any) error) error {
	var v any
	return into(b, &v)
}

// A pass-through any argument statically holds whatever pointer the caller
// passed; the signature is noanyparams' report, not an untyped target.
func decodeJSON(b []byte, v any) error { return json.Unmarshal(b, v) }

func decodeInto(r *bytes.Reader, dst any) error { return json.NewDecoder(r).Decode(dst) }

// A *any parameter is an untyped target even without the & at the call site.
func intoAnyPtr(b []byte, v *any) error {
	return json.Unmarshal(b, v) // want `nountypedunmarshal: json\.Unmarshal into any keeps the document untyped`
}

// A same-package named pointer type is resolved.
type anyPtr *any

func intoNamedAnyPtr(b []byte, v anyPtr) error {
	return json.Unmarshal(b, v) // want `nountypedunmarshal: json\.Unmarshal into any keeps the document untyped`
}
