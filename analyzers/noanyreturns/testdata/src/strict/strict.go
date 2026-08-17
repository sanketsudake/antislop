package strict

// container/heap.Interface dictates Pop() any, but allow-dictated is off here.
type Heap []int

func (h *Heap) Pop() any { return nil } // want `noanyreturns: result has type any`
