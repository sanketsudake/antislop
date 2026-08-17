package a

import (
	"testing"
	"time"
)

// --- invalid ---

func TestPatchedClock(t *testing.T) {
	timeNow = func() time.Time { return time.Time{} } // want `nomonkeypatch: test reassigns package-level function variable "timeNow"; pass the dependency in through the constructor or a parameter instead of patching the package`
	_ = timeNow()
}

// --- valid ---

// A struct field is the injection seam, not a patch.
func TestFieldInjection(t *testing.T) {
	s := server{now: time.Now}
	s.now = func() time.Time { return time.Time{} }
	_ = s.at()
}

// A local function variable belongs to this test alone.
func TestLocalVar(t *testing.T) {
	now := time.Now
	now = func() time.Time { return time.Time{} }
	_ = now()
}

// A package-level variable that is not a function is state, not a call site.
func TestPlainVar(t *testing.T) {
	verbose = true
	_ = verbose
}

// A declaration is not a reassignment.
func TestShadow(t *testing.T) {
	timeNow := time.Now
	_ = timeNow()
}
