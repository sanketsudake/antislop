// Package lib stands in for a third-party package that hands out untyped values.
package lib

// Claims is a named map of untyped values, like jwt.MapClaims.
type Claims map[string]any

// Event carries an untyped payload, like cache.DeletedFinalStateUnknown.Obj.
type Event struct {
	Obj any
}

// Handler is a callback slot whose parameter type the implementer cannot change.
type Handler struct {
	OnEvent func(ev any)
}

// Object holds an untyped document, like unstructured.Unstructured.Object.
type Object struct {
	Object map[string]any
}
