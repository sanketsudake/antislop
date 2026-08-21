//go:build go1.27

// Package go127 pins the advice that only a Go 1.27 file can act on: a
// concrete method may declare its own type parameters, so the diagnostic
// offers that alongside the named type. The build tag both keeps the file out
// of a Go 1.26 build and sets the file's language version, which is what the
// analyzer reads.
package go127

type Box struct{ v int }

// A concrete method: the advice names the method type parameter.
func (b Box) Take(x any) { _ = x } // want `parameter "x" has type any.*or give the method its own type parameter \(Go 1\.27\) so the caller's type survives the call`

// A method that already declares type parameters keeps the plain advice: the
// type parameter is there and the any is a separate decision.
func (b Box) Mixed[T any](t T, x any) T { _ = x; return t } // want `parameter "x" has type any.*decode input at its I/O boundary into a named type and accept that type$`

// A plain function is not a method: generic functions long predate Go 1.27,
// so nothing changes.
func Plain(x any) { _ = x } // want `parameter "x" has type any.*accept that type$`

// An interface method may not declare type parameters, nor may a generic
// method implement one, so it keeps the plain advice.
type Sink interface {
	Put(x any) // want `parameter "x" has type any.*accept that type$`
}

// A func literal is not a method declaration.
var lit = func(x any) { _ = x } // want `parameter "x" has type any.*accept that type$`

// A func type in a field position is not a method declaration.
type Holder struct {
	Fn func(x any) // want `parameter "x" has type any.*accept that type$`
}

// A generic method with no any parameter is clean.
func (b Box) Map[T any](f func(int) T) T { return f(b.v) }
