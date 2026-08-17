package example

import "fmt"

// Rejected by safetycomment.
func FirstName(values []fmt.Stringer) string {
	return values[0].(interface{ Name() string }).Name()
}

// Accepted: the invariant that makes the assertion valid is written down, so
// the next reader can check it.
func FirstNameJustified(values []fmt.Stringer) string {
	// SAFETY: NamedValues is the only constructor of this slice and every
	// element it produces has a Name method.
	return values[0].(interface{ Name() string }).Name()
}

// Accepted: the comma-ok form states the check in code instead of in prose.
func FirstNameOrEmpty(values []fmt.Stringer) (string, bool) {
	named, ok := values[0].(interface{ Name() string })
	if !ok {
		return "", false
	}
	return named.Name(), true
}
