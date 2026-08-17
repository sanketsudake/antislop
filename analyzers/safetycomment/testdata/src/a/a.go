package a

import (
	"fmt"
	"unsafe"
)

type MyErr struct{ msg string }

func (e *MyErr) Error() string { return e.msg }

type Header struct {
	Kind  uint8
	Count uint32
}

var (
	raw     any
	failure error
)

// --- invalid: escape hatches without a stated invariant ------------------------

func plain() int {
	return raw.(int) // want `safetycomment: unchecked type assertion has no SAFETY: justification`
}

// A concrete interface operand panics just the same.
func concrete() string {
	return failure.(*MyErr).msg // want `safetycomment: unchecked type assertion has no SAFETY: justification`
}

func inExpression() int {
	return raw.(int) + 1 // want `safetycomment: unchecked type assertion has no SAFETY`
}

func asArgument() string {
	return fmt.Sprint(raw.(int)) // want `safetycomment: unchecked type assertion has no SAFETY`
}

// A comment after the assertion states nothing at the point of the assertion.
func trailing() int {
	return raw.(int) // want `safetycomment: unchecked type assertion has no SAFETY`
	// SAFETY: this comment arrives after the fact.
}

// SAFETY: a doc comment on the function is too far from the assertion to say
// which invariant this one line relies on.
func docOnFunc() int {
	return raw.(int) // want `safetycomment: unchecked type assertion has no SAFETY`
}

// A one-line body does not bring the doc comment into reach: the justification
// must live inside the function it justifies.
// SAFETY: a doc comment on a one-line body is still a doc comment.
func docOnOneLiner() int { return raw.(int) } // want `safetycomment: unchecked type assertion has no SAFETY`

// A justification above the enclosing function literal is out of reach too.
func aboveLiteral() func() int {
	// SAFETY: too far: the assertion lives in another scope.
	return func() int {
		return raw.(int) // want `safetycomment: unchecked type assertion has no SAFETY`
	}
}

// The same holds when the literal is written on one line.
func aboveInlineLiteral() func() int {
	// SAFETY: this justifies the return, not the assertion inside the literal.
	return func() int { return raw.(int) } // want `safetycomment: unchecked type assertion has no SAFETY`
}

func toPointer(x *int) unsafe.Pointer {
	return unsafe.Pointer(x) // want `safetycomment: unsafe conversion has no SAFETY: justification`
}

func fromPointer(p unsafe.Pointer) *int {
	return (*int)(p) // want `safetycomment: unsafe conversion has no SAFETY: justification`
}

func toUintptr(p unsafe.Pointer) uintptr {
	return uintptr(p) // want `safetycomment: unsafe conversion has no SAFETY`
}

func slice(p *byte, n int) []byte {
	return unsafe.Slice(p, n) // want `safetycomment: unsafe conversion has no SAFETY`
}

func str(p *byte, n int) string {
	return unsafe.String(p, n) // want `safetycomment: unsafe conversion has no SAFETY`
}

func data(b []byte) *byte {
	return unsafe.SliceData(b) // want `safetycomment: unsafe conversion has no SAFETY`
}

func stringData(s string) *byte {
	return unsafe.StringData(s) // want `safetycomment: unsafe conversion has no SAFETY`
}

func advance(p unsafe.Pointer) unsafe.Pointer {
	return unsafe.Add(p, 4) // want `safetycomment: unsafe conversion has no SAFETY`
}

// --- valid ---------------------------------------------------------------------

// A justification on the statement above the assertion.
func justified() int {
	// SAFETY: raw is filled by decode() below, which only ever stores an int.
	return raw.(int)
}

// The justification may run over several lines; the group is what counts.
func justifiedGroup() int {
	// SAFETY: raw is filled by decode() below,
	// which only ever stores an int.
	return raw.(int)
}

// An inline justification immediately before the expression.
func inline() int {
	return /* SAFETY: raw is set by decode() to an int */ raw.(int)
}

// The owning statement carries the justification even when the assertion sits
// deeper inside it.
func nested() map[string]int {
	// SAFETY: raw is set by decode() to an int.
	return map[string]int{"n": raw.(int)}
}

// The if statement owns its initialiser.
func inIf() bool {
	// SAFETY: raw is set by decode() to an int.
	if v := raw.(int); v > 0 {
		return true
	}
	return false
}

// A declaration is a statement too.
// SAFETY: raw is set by decode() to an int before this package is used.
var cached = raw.(int)

var (
	// SAFETY: raw is set by decode() to an int before this package is used.
	cachedAgain = raw.(int)
)

// The checked forms state the invariant in code, so no comment is needed.
func commaOK() (int, bool) {
	n, ok := raw.(int)
	return n, ok
}

func typeSwitch() string {
	switch raw.(type) {
	case int:
		return "int"
	}
	return "other"
}

// Compile-time unsafe queries move no pointers.
func sizes(h Header) uintptr {
	return unsafe.Sizeof(h) + unsafe.Alignof(h) + unsafe.Offsetof(h.Count)
}

func justifiedPointer(x *int) unsafe.Pointer {
	// SAFETY: x is a live pointer owned by the caller for the whole call.
	return unsafe.Pointer(x)
}

func justifiedSlice(p *byte, n int) []byte {
	// SAFETY: p points at n bytes allocated by the caller and kept alive by it.
	return unsafe.Slice(p, n)
}

// An ordinary conversion is not an escape hatch.
func ordinary(s string) []byte { return []byte(s) }
