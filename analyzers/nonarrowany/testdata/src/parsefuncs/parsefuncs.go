package parsefuncs

import "fmt"

type User struct{ Name string }

// --- valid with allow-in-parse-funcs: boundary functions that decode ------------

// The database/sql Scanner contract: an any input and an error result.
type NullName struct{ Name string }

func (n *NullName) Scan(src any) error {
	switch v := src.(type) {
	case string:
		n.Name = v
		return nil
	}
	return fmt.Errorf("unsupported")
}

// A bool result marks a predicate boundary too.
func parseUser(v any) (User, error) {
	u, ok := v.(User)
	if !ok {
		return User{}, fmt.Errorf("not a user")
	}
	return u, nil
}

func isName(v any) bool {
	_, ok := v.(string)
	return ok
}

// A func literal with the same shape is a boundary as well.
var decode = func(v any) error {
	switch v.(type) {
	case string:
		return nil
	}
	return fmt.Errorf("unsupported")
}

// --- still invalid: narrowing outside a boundary function -----------------------

// No error or bool result: this is not a decode boundary, it is a dispatcher.
func describe(v any) string {
	switch v.(type) { // want `nonarrowany: type switch on any narrows a representation`
	case int:
		return "int"
	}
	return "other"
}

// An error result without an any parameter narrows a value that some other
// boundary should already have decoded.
var stored any

func check() error {
	if _, ok := stored.(string); ok { // want `nonarrowany: comma-ok assertion on any narrows a representation`
		return nil
	}
	return fmt.Errorf("bad")
}
