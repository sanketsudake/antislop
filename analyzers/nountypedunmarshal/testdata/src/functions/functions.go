package functions

import (
	"decoder"
	"encoding/json"
)

// --- invalid ---

func intoAny(b []byte) error {
	var v any
	return decoder.Decode(b, &v) // want `nountypedunmarshal: decoder\.Decode into any keeps the document untyped; decode into a struct that names the fields you use`
}

// --- valid ---

// The functions option replaced the list, so encoding/json is no longer in it.
func jsonIntoAny(b []byte) error {
	var v any
	return json.Unmarshal(b, &v)
}
