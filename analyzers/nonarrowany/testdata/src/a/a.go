package a

import (
	"database/sql/driver"
	"io"
	"net"
)

type Escape = any

var (
	raw     any
	boxed   Escape
	value   driver.Value
	failure error
	reader  io.Reader
)

// --- invalid: checked narrowing of an empty-interface value --------------------

func commaOK() (int, bool) {
	n, ok := raw.(int) // want `nonarrowany: comma-ok assertion on any narrows a representation without establishing its contract`
	return n, ok
}

func commaOKAssign() bool {
	var ok bool
	_, ok = raw.(string) // want `nonarrowany: comma-ok assertion on any narrows a representation`
	return ok
}

func commaOKSpec() bool {
	var s, ok = raw.(string) // want `nonarrowany: comma-ok assertion on any narrows a representation`
	_ = s
	return ok
}

func switchPlain() string {
	switch raw.(type) { // want `nonarrowany: type switch on any narrows a representation without establishing its contract`
	case int:
		return "int"
	}
	return "other"
}

func switchBinding() string {
	switch v := raw.(type) { // want `nonarrowany: type switch on any narrows a representation`
	case int:
		_ = v
		return "int"
	}
	return "other"
}

// A same-package alias of any is still any.
func aliased() bool {
	_, ok := boxed.(int) // want `nonarrowany: comma-ok assertion on any narrows a representation`
	return ok
}

// An imported empty interface (driver.Value) is untyped all the same: the smell
// is narrowing a value that carries no evidence, wherever the name came from.
func imported() string {
	switch value.(type) { // want `nonarrowany: type switch on any narrows a representation`
	case int64:
		return "int64"
	}
	return "other"
}

// Parentheses do not hide the operand.
func parenthesised() bool {
	_, ok := (raw).(int) // want `nonarrowany: comma-ok assertion on any narrows a representation`
	return ok
}

// --- valid ---------------------------------------------------------------------

// A single-value assertion is safetycomment's domain, not this rule's.
func unchecked() int {
	// SAFETY: the fixture only needs the shape, not a real invariant.
	return raw.(int)
}

// Narrowing a non-empty interface tests a contract the type already carries.
func netError() bool {
	_, ok := failure.(net.Error)
	return ok
}

func readerSwitch() string {
	switch r := reader.(type) {
	case *net.TCPConn:
		_ = r
		return "conn"
	}
	return "other"
}

// The generics idiom: any(v) exists only because the language has no other way
// to switch on a type parameter.
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

func typeParamInterfaceLit[T any](v T) bool {
	_, ok := interface{}(v).(int)
	return ok
}

// A non-empty interface conversion is not an any hop.
func readerConversion(f *net.TCPConn) bool {
	_, ok := io.Reader(f).(io.Closer)
	return ok
}
