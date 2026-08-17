package a

import "database/sql/driver"

// --- invalid: declarations that rename the empty interface --------------------

type Payload = any // want `noanytypes: type "Payload" is the empty interface under another name`

type Blob any // want `noanytypes: type "Blob" is the empty interface under another name`

type Raw interface{} // want `noanytypes: type "Raw" is the empty interface under another name`

type Embedded interface{ any } // want `noanytypes: type "Embedded" is the empty interface under another name`

// Renaming a local empty interface transitively is still renaming it, and each
// declaration in the chain is reported at its own name.
type Chained Blob // want `noanytypes: type "Chained" is the empty interface under another name`

type ChainedAlias = Chained // want `noanytypes: type "ChainedAlias" is the empty interface under another name`

// A generic alias whose right-hand side resolves to the empty interface.
type Box[T any] = any // want `noanytypes: type "Box" is the empty interface under another name`

func local() {
	type Inner any // want `noanytypes: type "Inner" is the empty interface under another name`
	var _ Inner
}

// --- valid --------------------------------------------------------------------

// A constraint interface is a type set, not the empty interface.
type Comparable interface{ comparable }

type Number interface{ ~int | ~float64 }

// An interface with methods states what the value can do.
type Reader interface{ Read(p []byte) (int, error) }

// A type parameter is not the empty interface, even when its constraint is any.
func generic[T any]() {
	type Same = T
	var _ Same
}

type SliceOf[T any] = []T

// The imported type owns its contract.
type Value = driver.Value

type Record struct{ ID string }

type IDs []string

type Decode func(data []byte) (Record, error)
