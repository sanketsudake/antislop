package a

import (
	"fmt"

	"foreign"
)

type Escape = any

type Boxed interface{}

type User struct{ ID string }

func (u User) String() string { return u.ID }

type Event struct {
	Payload any
	Name    string
}

var (
	pkgAny any
	errv   error
)

func pair() (int, string) { return 1, "x" }

func sink(v any) {}

// --- invalid: a known concrete value stored where nothing states its type ----

// (a) a var declared with an empty-interface type and a concrete initializer.
var declared any = 42 // want `noknownwidening: value of type int is stored as any`

var literal interface{} = "s" // want `noknownwidening: value of type string is stored as any`

// A same-package alias and a same-package named type are the same location.
var aliased Escape = 1.5 // want `noknownwidening: value of type float64 is stored as any`

var boxedUser Boxed = User{} // want `noknownwidening: value of type User is stored as any`

// Each name/value pair is judged on its own.
var one, two any = 1, "x" // want `noknownwidening: value of type int is stored as any` `noknownwidening: value of type string is stored as any`

// A constructor call is as concrete as a literal.
var built any = User{ID: "u1"} // want `noknownwidening: value of type User is stored as any`

func localSpec() {
	var local any = 42 // want `noknownwidening: value of type int is stored as any`
	_ = local
}

// (b) an assignment whose location is typed as the empty interface.
func assignments(e *Event, u User) {
	pkgAny = 42 // want `noknownwidening: value of type int is stored as any`
	var local any
	local = "s" // want `noknownwidening: value of type string is stored as any`
	_ = local
	e.Payload = u          // want `noknownwidening: value of type User is stored as any`
	pkgAny, local = 1, "x" // want `noknownwidening: value of type int is stored as any` `noknownwidening: value of type string is stored as any`
}

// (c) an explicit conversion to the empty interface.
func conversions(u User) {
	_ = any(42)        // want `noknownwidening: value of type int is stored as any`
	_ = interface{}(u) // want `noknownwidening: value of type User is stored as any`
	_ = Escape(1.5)    // want `noknownwidening: value of type float64 is stored as any`
	sink(any(42))      // want `noknownwidening: value of type int is stored as any`
	_ = (any(u))       // want `noknownwidening: value of type User is stored as any`
}

// --- valid -------------------------------------------------------------------

// No initializer: nothing is discarded yet.
var empty any

// The target is not the empty interface, so the evidence survives.
var stringer fmt.Stringer = User{}

// Another package owns this contract; its use site is not our decision.
var foreignValue foreign.Value = 42

// The initializer is already an interface value: nothing concrete is lost.
var fromIface any = errv

// A tuple-returning call has no per-value pairing to judge.
var first, second any = pair()

func validAssignments(e *Event, m map[string]any, xs []any) {
	// Containers are noanycontainers' report, not this one.
	m["k"] = 42
	xs[0] = 42
	// The field of a composite literal is noanyfields' report.
	_ = Event{Payload: 42}
	// A concrete value passed to a parameter typed any is the signature's
	// problem: noanyparams owns it.
	sink(42)
	fmt.Println(42)
	// The location is not the empty interface.
	e.Name = "n"
	_ = 42
}

// A result typed any is noanyreturns' report.
func returnsAny() any { return 42 }

func validConversions(u User) {
	// Forwarding to a variadic ...any parameter is the fmt idiom.
	fmt.Println(any(42))
	// So is building the []any that such an API takes.
	_ = []any{any(1)}
	// The operand is already an interface value.
	_ = any(errv)
	// Asserting on the conversion is nochainedassert's report.
	_ = any(u).(User)
	switch any(u).(type) {
	case User:
	}
	// A conversion to a non-empty interface keeps the evidence.
	_ = fmt.Stringer(u)
}

// any(v) on a type parameter is the generics idiom: the language offers no
// other way to inspect T.
func generic[T any](v T) {
	_ = any(v)
	sink(any(v))
}
