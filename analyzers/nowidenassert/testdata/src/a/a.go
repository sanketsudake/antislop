package a

import "fmt"

type User struct{ ID string }

func (u User) String() string { return u.ID }

type Box struct{ Payload any }

var (
	pkgAny any = 42
	errv   error
)

func raw() any { return 42 }

// --- invalid: widened here, asserted back in the same function ---------------

func declaredThenAsserted() int {
	var boxed any = 42
	return boxed.(int) // want `nowidenassert: "boxed" was widened to any at line \d+ and is asserted back here`
}

// The conversion form of the same mistake.
func convertedThenAsserted(u User) User {
	boxed := any(u)
	return boxed.(User) // want `nowidenassert: "boxed" was widened to any at line \d+ and is asserted back here`
}

// var v = any(x) is the same declaration without the written type.
func inferredThenAsserted() int {
	var boxed = any(42)
	return boxed.(int) // want `nowidenassert: "boxed" was widened to any at line \d+ and is asserted back here`
}

// The comma-ok form checks what the function already knew.
func checkedAgain() (int, bool) {
	var boxed any = 42
	v, ok := boxed.(int) // want `nowidenassert: "boxed" was widened to any at line \d+ and is asserted back here`
	return v, ok
}

// A type switch with a case for the known type is the same round trip.
func switchedBack() string {
	var boxed any = 42
	switch boxed.(type) { // want `nowidenassert: "boxed" was widened to any at line \d+ and is asserted back here`
	case string:
		return "string"
	case int:
		return "int"
	}
	return "other"
}

// Asserting to an interface the concrete type already implements is the same
// round trip: the compiler could have seen it.
func assertedToImplemented(u User) fmt.Stringer {
	var boxed any = u
	return boxed.(fmt.Stringer) // want `nowidenassert: "boxed" was widened to any at line \d+ and is asserted back here`
}

// A closure that only reads the variable does not undo the round trip in the
// function that declared it.
func widenedBesideAClosure() int {
	var boxed any = 42
	inner := func() int { return boxed.(int) }
	return boxed.(int) + inner() // want `nowidenassert: "boxed" was widened to any at line \d+ and is asserted back here`
}

// --- valid -------------------------------------------------------------------

// Reassigned in between: what the declaration knew no longer holds.
func reassigned(other any) int {
	var boxed any = 42
	boxed = other
	return boxed.(int)
}

// Its address escaped, so any callee may have replaced the value.
func addressTaken() int {
	var boxed any = 42
	store(&boxed)
	return boxed.(int)
}

func store(p *any) { *p = 7 }

// A package-level variable is written from anywhere.
func packageLevel() int {
	return pkgAny.(int)
}

// The variable belongs to the enclosing function, so this function did not
// widen anything.
func closure() func() int {
	var boxed any = 42
	return func() int { return boxed.(int) }
}

// The asserted type is unrelated to what was widened: this is a real question,
// not a round trip.
func unrelatedType() string {
	var boxed any = 42
	if s, ok := boxed.(string); ok {
		return s
	}
	return ""
}

// A type switch with no case for the known type narrows nothing back.
func switchedElsewhere() string {
	var boxed any = 42
	switch boxed.(type) {
	case string:
		return "string"
	}
	return "other"
}

// The initializer is already an interface value: nothing concrete was widened.
func fromInterface() string {
	var boxed any = errv
	if s, ok := boxed.(fmt.Stringer); ok {
		return s.String()
	}
	return ""
}

// The declaration carries no value at all.
func noInitializer(v int) int {
	var boxed any
	boxed = v
	return boxed.(int)
}

// The operand is not an identifier, so there is no declaration to point at.
func notAnIdent(b Box) int {
	return raw().(int) + b.Payload.(int)
}

// The variable is not the empty interface, so its type still says something.
func nonEmptyTarget(u User) User {
	var s fmt.Stringer = u
	return s.(User)
}

// A type parameter is not a concrete type, so nothing known was widened.
func generic[T any](v T) int {
	var boxed any = v
	return boxed.(int)
}
