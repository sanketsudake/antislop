package a

import (
	"container/heap"
	"database/sql/driver"
	"sync"
)

// --- invalid: results typed empty interface -----------------------------------

func Load() any { return nil } // want `noanyreturns: result has type any`

func LoadPair() (any, error) { return nil, nil } // want `noanyreturns: result has type any`

func Named() (v any) { return nil } // want `noanyreturns: result "v" has type any`

func Two() (a, b any) { return nil, nil } // want `noanyreturns: results "a", "b" have type any`

func Bare() interface{} { return nil } // want `noanyreturns: result has type interface\{\}`

type Payload = any

type Blob interface{}

func Alias() Payload { return nil } // want `noanyreturns: result has type Payload`

func Local() Blob { return nil } // want `noanyreturns: result has type Blob`

// A func type in a type position is a contract too, and is never dictated.
type Producer func() any // want `noanyreturns: result has type any`

type Options struct {
	Decode func() (any, error) // want `noanyreturns: result has type any`
}

var decode func() any // want `noanyreturns: result has type any`

// Own interface: the contract is reported once, at the interface.
type Loader interface {
	Load() any // want `noanyreturns: result has type any`
}

// Implementer of the same-package interface is not reported again.
type fileLoader struct{}

func (fileLoader) Load() any { return nil }

func literal() {
	fn := func() any { return nil } // want `noanyreturns: result has type any`
	_ = fn
}

// --- valid --------------------------------------------------------------------

// Type parameters are not any.
func Zero[T any]() T { var z T; return z }

// Imported named type is the other package's contract.
func Convert(v driver.Value) driver.Value { return v }

// Interface-dictated: container/heap.Interface (direct import).
type IntHeap []int

func (h IntHeap) Len() int           { return len(h) }
func (h IntHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h IntHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *IntHeap) Push(x any)        { *h = append(*h, x.(int)) }
func (h *IntHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

var _ heap.Interface = (*IntHeap)(nil)

// Well-known contract without importing the yaml package: MarshalYAML() (any, error).
type Node struct{ Name string }

func (n *Node) MarshalYAML() (any, error) { return n.Name, nil }

// Well-known contract without importing flag: Get() any.
type counter int

func (c counter) Get() any { return int(c) }

// Func literal in an imported struct field: sync.Pool.New dictates func() any.
var pool = sync.Pool{New: func() any { return new(int) }}

// A nested func type inside a dictated signature is reported once at the
// contract, not at the implementer.
type Supplier interface {
	With(fn func() any) // want `noanyreturns: result has type any`
}

type constSupplier struct{}

func (constSupplier) With(fn func() any) {}
