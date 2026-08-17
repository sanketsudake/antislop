package example

// raw holds whatever the decoder handed back, so nothing states what it is.
var raw any

// Rejected by nonarrowany.
func RawPort() (int, bool) { port, ok := raw.(int); return port, ok }

// Rejected by nonarrowany: a type switch is the same test in another shape.
func RawKind() string {
	switch raw.(type) {
	case int:
		return "int"
	}
	return "other"
}

// Accepted: decode once at the boundary into a named type, then branch on the
// type instead of on the representation.
type Listener struct {
	Port int
	Kind string
}

func (l Listener) Describe() string { return l.Kind }

func ListenerPort(l Listener) (int, bool) { return l.Port, l.Port != 0 }
