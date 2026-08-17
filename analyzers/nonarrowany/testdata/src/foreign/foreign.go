package foreign

import "foreign/lib"

// Elements of a foreign named container of any: the library's contract.
func claimsExp(c lib.Claims) (float64, bool) {
	exp, ok := c["exp"].(float64)
	return exp, ok
}

// A field of a foreign struct typed any: the library's contract.
func eventObj(e lib.Event) (string, bool) {
	s, ok := e.Obj.(string)
	return s, ok
}

// A parameter of a func literal in a foreign callback slot: dictated.
var handler = lib.Handler{OnEvent: func(ev any) {
	switch ev.(type) {
	case string:
	}
}}

// A same-package container of any is ours to fix, so its elements are still
// reported (noanycontainers reports the declaration too).
type ownClaims map[string]any

func ownExp(c ownClaims) (float64, bool) {
	exp, ok := c["exp"].(float64) // want `nonarrowany: comma-ok assertion on any`
	return exp, ok
}

// A same-package field typed any is ours to fix.
type ownEvent struct{ Obj any }

func ownObj(e ownEvent) (string, bool) {
	s, ok := e.Obj.(string) // want `nonarrowany: comma-ok assertion on any`
	return s, ok
}

// A parameter of our own (non-dictated) function is ours to fix.
func ownParam(ev any) bool {
	_, ok := ev.(string) // want `nonarrowany: comma-ok assertion on any`
	return ok
}

// A foreign untyped value reached through a local variable, or indexed off a
// foreign field typed as a container of any, is still the library's contract.
func viaVariable(c lib.Claims) (float64, bool) {
	raw, present := c["exp"]
	if !present {
		return 0, false
	}
	exp, ok := raw.(float64)
	return exp, ok
}

func offForeignField(o lib.Object) (string, bool) {
	kind, ok := o.Object["kind"].(string)
	return kind, ok
}
