package example

// Rejected by nochainedassert.
func Count(v int) int {
	// SAFETY: v was widened to any on the line itself, so it is still an int.
	return any(v).(int)
}

// Accepted: assert nothing; v already has the type the caller needs.
func CountDirect(v int) int { return v }

// Rejected by nochainedassert: walking a decoded document one assertion at a
// time (noanycontainers also reports each map[string]any).
var payload any

func Width() int {
	// SAFETY: the document was validated against the viewport schema on decode.
	return payload.(map[string]any)["viewport"].(map[string]any)["width"].(int)
}

// Accepted: decode the document into a struct once.
type Viewport struct{ Width int }

type Payload struct{ Viewport Viewport }

func WidthTyped(p Payload) int { return p.Viewport.Width }

// Accepted: any(v) on a type parameter is the generics idiom, not a hop
// through any: the language offers no other way to switch on T.
func KindOf[T any](v T) string {
	switch any(v).(type) {
	case int:
		return "int"
	}
	return "other"
}
