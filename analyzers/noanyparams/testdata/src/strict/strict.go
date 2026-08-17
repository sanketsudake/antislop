package strict

func Logf(format string, args ...any) {} // want `noanyparams: parameter "args" has type any`

type Heap []int

func (h *Heap) Push(x any) {} // want `noanyparams: parameter "x" has type any`
