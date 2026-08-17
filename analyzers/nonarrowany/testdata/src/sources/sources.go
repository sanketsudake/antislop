package sources

import (
	"container/heap"
	"container/list"
	"context"
	"sync"
	"sync/atomic"
)

type key struct{}

// Narrowing the immediate result of a standard-library API that returns an
// untyped value is the boundary of that value: not reported by default.

func fromContext(ctx context.Context) (string, bool) {
	ns, ok := ctx.Value(key{}).(string)
	return ns, ok
}

func fromContextViaVar(ctx context.Context) (string, bool) {
	v := ctx.Value(key{})
	s, ok := v.(string)
	return s, ok
}

func fromSyncMap(m *sync.Map, k string) (int, bool) {
	if v, ok := m.Load(k); ok {
		n, isInt := v.(int)
		return n, isInt
	}
	return 0, false
}

func fromSyncMapSwitch(m *sync.Map, k string) string {
	v, _ := m.LoadAndDelete(k)
	switch v.(type) {
	case int:
		return "int"
	}
	return "other"
}

func fromAtomic(av *atomic.Value) (int, bool) {
	n, ok := av.Load().(int)
	return n, ok
}

func fromListElement(e *list.Element) (int, bool) {
	n, ok := e.Value.(int)
	return n, ok
}

type intHeap []int

func (h intHeap) Len() int           { return len(h) }
func (h intHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h intHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *intHeap) Push(x any)        { *h = append(*h, x.(int)) }
func (h *intHeap) Pop() any          { old := *h; x := old[len(old)-1]; *h = old[:len(old)-1]; return x }

func fromHeap(h *intHeap) (int, bool) {
	n, ok := heap.Pop(h).(int)
	return n, ok
}

// A value that did not come from a listed source is still reported.

var raw any

func fromRaw() (int, bool) {
	n, ok := raw.(int) // want `nonarrowany: comma-ok assertion on any`
	return n, ok
}

// A variable that is reassigned after the source call is no longer the
// source's immediate result.
func reassigned(ctx context.Context) (int, bool) {
	v := ctx.Value(key{})
	v = raw
	n, ok := v.(int) // want `nonarrowany: comma-ok assertion on any`
	return n, ok
}

// --- foreign contracts -------------------------------------------------------

// A parameter of a function whose signature is dictated (a callback slot typed
// by another package, an interface method) cannot be retyped; narrowing it is
// the boundary of that contract.
type Model struct {
	Step func(st, in any) bool // the contract; noanyparams reports it here
}

var model = Model{Step: func(st, in any) bool {
	_, ok := st.(int)
	return ok
}}

func dictatedLiteral(m *sync.Map) {
	m.Range(func(k, v any) bool {
		_, ok := v.(int)
		return ok
	})
}

// A field of a foreign struct typed any, and an element of a foreign named
// container of any, are the other package's contract; narrowing them at the
// point of receipt is the boundary too.
func foreignField(e *list.Element) bool {
	_, ok := e.Value.(int) // list.Element.Value: also on the sources list, but a foreign field regardless
	return ok
}
