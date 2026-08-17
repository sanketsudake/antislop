package example

// Rejected by nostructuralnames.
type OrderShape struct{ ID string }

// Accepted: the name states the role the value plays in the domain.
type Order struct{ ID string }
