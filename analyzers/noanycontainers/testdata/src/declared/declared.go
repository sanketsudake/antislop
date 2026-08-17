package declared

// With skip-declared-any, a container literal that fills a slot declared in
// this package -- a result, a parameter, a field, a typed variable -- is left
// to the finding at that declaration.

func build() map[string]any { // want `noanycontainers: map\[string\]any erases the value types`
	return map[string]any{"a": 1}
}

func consume(m map[string]any) {} // want `noanycontainers: map\[string\]any erases the value types`

func callOwn() {
	consume(map[string]any{"a": 1})
}

type holder struct {
	Meta map[string]any // want `noanycontainers: map\[string\]any erases the value types`
}

func fill(h *holder) {
	h.Meta = map[string]any{"a": 1}
	_ = holder{Meta: map[string]any{"b": 2}}
}

var typed map[string]any = map[string]any{"a": 1} // want `noanycontainers: map\[string\]any erases the value types`

// A literal that declares its own type is the declaration, and is reported.
func fresh() {
	m := map[string]any{"a": 1} // want `noanycontainers: map\[string\]any erases the value types`
	_ = m
	var v = map[string]any{"a": 1} // want `noanycontainers: map\[string\]any erases the value types`
	_ = v
}

// A literal nested inside a skipped literal is part of the same document.
func nested() map[string]any { // want `noanycontainers: map\[string\]any erases the value types`
	return map[string]any{"a": map[string]any{"b": []map[string]any{{"c": 1}}}}
}
