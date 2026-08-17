package foreign

import (
	"foreign/lib"
	"sync"
)

// Receipts of a foreign contract need no SAFETY comment: the other package
// chose any, and the assertion is the boundary.

func claimsExp(c lib.Claims) float64 { return c["exp"].(float64) }

func eventObj(e lib.Event) string { return e.Obj.(string) }

var handler = lib.Handler{OnEvent: func(ev any) { _ = ev.(string) }}

func rangeValues(m *sync.Map) {
	m.Range(func(k, v any) bool { _ = v.(int); return true })
}

// Our own declarations are ours to fix; the assertion still needs SAFETY.

type ownClaims map[string]any

func ownExp(c ownClaims) float64 {
	return c["exp"].(float64) // want `safetycomment: unchecked type assertion has no SAFETY: justification`
}

type ownEvent struct{ Obj any }

func ownObj(e ownEvent) string {
	return e.Obj.(string) // want `safetycomment: unchecked type assertion has no SAFETY: justification`
}

func ownParam(ev any) string {
	return ev.(string) // want `safetycomment: unchecked type assertion has no SAFETY: justification`
}

func viaVariable(c lib.Claims) float64 {
	raw := c["exp"]
	return raw.(float64)
}

func offForeignField(o lib.Object) string { return o.Object["kind"].(string) }
