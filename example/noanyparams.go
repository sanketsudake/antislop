package example

import "container/heap"

// Rejected by noanyparams.
func Save(value any) {}

// Accepted: decode at the boundary into a named type.
type User struct{ ID string }

func SaveUser(user User) {}

// Accepted: the signature is dictated by container/heap.Interface.
type IntHeap []int

func (h IntHeap) Len() int           { return len(h) }
func (h IntHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h IntHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *IntHeap) Push(x any) {
	// SAFETY: heap.Interface.Push receives back the elements this package
	// pushed, and IntHeap only ever pushes ints.
	*h = append(*h, x.(int))
}
func (h *IntHeap) Pop() any { old := *h; x := old[len(old)-1]; *h = old[:len(old)-1]; return x }

var _ heap.Interface = (*IntHeap)(nil)
