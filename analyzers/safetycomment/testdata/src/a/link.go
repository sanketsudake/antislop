package a

import (
	_ "unsafe" // for go:linkname
)

//go:linkname runtimeNano runtime.nanotime // want `safetycomment: go:linkname has no SAFETY: justification`
func runtimeNano() int64

// SAFETY: runtime.nanotime is a stable monotonic clock read used only for
// timing, and this package keeps the signature in step with the runtime.
//
//go:linkname runtimeNanoAgain runtime.nanotime
func runtimeNanoAgain() int64
