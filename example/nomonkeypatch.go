package example

import "time"

// clock is the package-level seam that nomonkeypatch_test.go replaces.
var clock = time.Now

// Accepted: the dependency arrives through a field the caller fills in, so
// the substitution is written in the type instead of in a global.
type Recorder struct {
	now func() time.Time
}

func NewRecorder(now func() time.Time) Recorder { return Recorder{now: now} }

func (r Recorder) At() time.Time { return r.now() }
