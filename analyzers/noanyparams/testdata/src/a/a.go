package a

import (
	"container/heap"
	"database/sql/driver"
	"fmt"
	"sync"
)

// --- invalid: parameters typed empty interface -------------------------------

func Save(v any) {} // want `noanyparams: parameter "v" has type any`

func Store(key string, v interface{}) {} // want `noanyparams: parameter "v" has type interface\{\}`

func Pair(a, b any) {} // want `noanyparams: parameters "a", "b" have type any`

type Payload = any

type Blob interface{}

func Handle(p Payload) {} // want `noanyparams: parameter "p" has type Payload`

func HandleBlob(b Blob) {} // want `noanyparams: parameter "b" has type Blob`

func Unnamed(any) {} // want `noanyparams: parameter has type any`

// A func type in a type position is a contract too.
type Callback func(v any) // want `noanyparams: parameter "v" has type any`

type Options struct {
	OnEvent func(e any) // want `noanyparams: parameter "e" has type any`
}

// Own interface: the contract is reported once, at the interface.
type Sink interface {
	Write(v any) error // want `noanyparams: parameter "v" has type any`
}

// Implementer of the same-package interface is not reported again.
type fileSink struct{}

func (fileSink) Write(v any) error { return nil }

func literal() {
	fn := func(x any) {} // want `noanyparams: parameter "x" has type any`
	fn(1)
}

// --- valid --------------------------------------------------------------------

// Variadic ...any is allowed by default (fmt / slog idiom).
func Logf(format string, args ...any) { fmt.Printf(format, args...) }

// Type parameters are not any.
func Identity[T any](v T) T { return v }

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

// Well-known contract without importing database/sql: Scan(src any) error.
type NullName struct{ Valid bool }

func (n *NullName) Scan(src any) error { n.Valid = src != nil; return nil }

// Func literal in an imported struct field: sync.Pool.New dictates func() any,
// and here a func literal with an any parameter passed to an imported func.
var pool = sync.Pool{New: func() any { return new(int) }}

func useImported() {
	pool.Put(func(v any) {}) // slot typed by sync.Pool.Put(x any): dictated
}

// Package-level func referenced by name in a dictated slot.
func onEvent(e any) {}

var opts = struct{ Handler func(any) }{Handler: onEvent} // want `noanyparams: parameter has type any`

func run() { withHandler(onEvent) }

func withHandler(h HandlerFunc) {}

type HandlerFunc func(any) // want `noanyparams: parameter has type any`

// A nested func type inside a dictated signature is part of the contract that
// dictated it; it is reported once at the contract, not at the implementer.
type Unmarshaler interface {
	UnmarshalYAML(unmarshal func(any) error) error // want `noanyparams: parameter has type any`
}

type yamlConfig struct{}

func (c *yamlConfig) UnmarshalYAML(unmarshal func(any) error) error { return nil }
