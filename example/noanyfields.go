package example

// Rejected by noanyfields.
type Event struct {
	Payload any
}

// Accepted: the field states what it holds.
type EventPayload struct {
	Kind string
	Body []byte
}

type ParsedEvent struct {
	Payload EventPayload
}
