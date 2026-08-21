package strict

func Logf(format string, args ...any) {} // want `noanyparams: parameter "args" has type any`

type Heap []int

// The file states no language version, so the advice must end at the named
// type: the generic-method alternative is Go 1.27 wording, and the anchor
// fails the test if it is ever appended here.
func (h *Heap) Push(x any) {} // want `noanyparams: parameter "x" has type any.*accept that type$`
