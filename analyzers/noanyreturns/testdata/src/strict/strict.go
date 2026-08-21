package strict

// container/heap.Interface dictates Pop() any, but allow-dictated is off here.
type Heap []int

// The file states no language version, so the advice must end at the small
// interface: the generic-method alternative is Go 1.27 wording, and the
// anchor fails the test if it is ever appended here.
func (h *Heap) Pop() any { return nil } // want `noanyreturns: result has type any.*the caller needs\)$`
