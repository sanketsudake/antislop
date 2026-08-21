//go:build go1.27

// Package go127 pins the advice that only a Go 1.27 file can act on: a
// concrete method may declare its own type parameters, so the diagnostic
// offers that alongside the named type. The build tag both keeps the file out
// of a Go 1.26 build and sets the file's language version, which is what the
// analyzer reads.
package go127

type Box struct{ v int }

// A concrete method: the advice names the method type parameter.
func (b Box) Payload() any { return b.v } // want `result has type any.*or give the method its own type parameter \(Go 1\.27\) and return that`

// A method that already declares type parameters keeps the plain advice.
func (b Box) Mixed[T any](t T) (T, any) { return t, b.v } // want `result has type any.*a small interface with the methods the caller needs\)$`

// A plain function is not a method: generic functions long predate Go 1.27.
func Plain() any { return 1 } // want `result has type any.*the caller needs\)$`

// An interface method may not declare type parameters, nor may a generic
// method implement one, so it keeps the plain advice.
type Source interface {
	Fetch() any // want `result has type any.*the caller needs\)$`
}

// A func literal is not a method declaration.
var lit = func() any { return 1 } // want `result has type any.*the caller needs\)$`

// A func type in a field position is not a method declaration.
type Holder struct {
	Fn func() any // want `result has type any.*the caller needs\)$`
}

// A generic method with no any result is clean.
func (b Box) Map[T any](f func(int) T) T { return f(b.v) }
