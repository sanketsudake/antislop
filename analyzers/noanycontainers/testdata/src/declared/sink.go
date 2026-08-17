package declared

import "declared/ext"

// A literal passed to another package's function fills a slot we do not
// declare; nothing else reports it, so it is reported here.
func toForeign() {
	ext.Sink(map[string]any{"a": 1}) // want `noanycontainers: map\[string\]any erases the value types`
}
