package a

import (
	"fmt"
	"io"
	"net"
)

type Escape = any

type Named struct{ name string }

func (n Named) String() string { return n.name }

var (
	raw    any
	reader io.Reader
)

// --- invalid: assertions chained through the empty interface -------------------

func doubleAssert() int {
	return raw.(any).(int) // want `nochainedassert: assertion chained through any fabricates evidence`
}

func conversion(n Named) fmt.Stringer {
	return any(n).(fmt.Stringer) // want `nochainedassert: assertion chained through any fabricates evidence`
}

func interfaceLiteral(n *Named) *Named {
	return interface{}(n).(*Named) // want `nochainedassert: assertion chained through any fabricates evidence`
}

// A same-package alias of any is the same hop.
func aliased(n Named) fmt.Stringer {
	return Escape(n).(fmt.Stringer) // want `nochainedassert: assertion chained through any fabricates evidence`
}

// The checked forms fabricate the same evidence.
func commaOK(n int) (int, bool) {
	v, ok := any(n).(int) // want `nochainedassert: assertion chained through any fabricates evidence`
	return v, ok
}

func typeSwitch(n int) string {
	switch any(n).(type) { // want `nochainedassert: assertion chained through any fabricates evidence`
	case int:
		return "int"
	}
	return "other"
}

// Chaining through an assertion is reported whether or not the hop was any.
func throughAssertion() io.Closer {
	return reader.(io.ReadCloser).(io.Closer) // want `nochainedassert: assertion chained through any fabricates evidence`
}

// Only the outermost assertion of a chain is reported.
func chain(n int) string {
	return any(n).(any).(string) // want `nochainedassert: assertion chained through any fabricates evidence`
}

// Parentheses do not hide the hop.
func parenthesised(n Named) fmt.Stringer {
	return (any(n)).(fmt.Stringer) // want `nochainedassert: assertion chained through any fabricates evidence`
}

// --- valid ---------------------------------------------------------------------

// A single narrowing step asserts from the value's declared type.
func single() int {
	return raw.(int)
}

func singleSwitch() string {
	switch raw.(type) {
	case int:
		return "int"
	}
	return "other"
}

// The generics idiom: any(v) is the only way to switch on a type parameter.
func typeParamSwitch[T any](v T) string {
	switch any(v).(type) {
	case int:
		return "int"
	}
	return "other"
}

func typeParamCommaOK[T any](v T) bool {
	_, ok := any(v).(int)
	return ok
}

// A conversion to a non-empty interface keeps the evidence it had.
func nonEmptyConversion(c *net.TCPConn) io.Closer {
	return io.Reader(c).(io.Closer)
}

// An ordinary conversion before a call is not a hop.
func ordinary(s fmt.Stringer) string {
	return fmt.Sprint(s)
}

// --- walking an untyped document: assertion, index, assertion ---------------

var doc any

func walkMap() string {
	return doc.(map[string]any)["value"].(string) // want `nochainedassert: assertion chained through an any-valued index walks an untyped document`
}

func walkDeep() string {
	// The outermost step is reported once for the whole chain.
	return doc.(map[string]any)["a"].(map[string]any)["b"].(string) // want `nochainedassert: assertion chained through an any-valued index walks an untyped document`
}

func walkSlice() string {
	return doc.([]any)[0].(string) // want `nochainedassert: assertion chained through an any-valued index walks an untyped document`
}

func walkCommaOK() (string, bool) {
	s, ok := doc.(map[string]any)["value"].(string) // want `nochainedassert: assertion chained through an any-valued index walks an untyped document`
	return s, ok
}

// A single index into a plain map of any is one narrowing step, not a chain
// (noanycontainers reports the map type).
var plain map[string]any

func singleIndex() string {
	return plain["k"].(string)
}

// Indexing a typed container after an assertion is not a walk: the element
// type carries evidence.
func typedIndex() int {
	return doc.([]int)[0]
}
