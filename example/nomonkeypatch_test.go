package example

import (
	"testing"
	"time"
)

// Rejected by nomonkeypatch.
func TestClock(t *testing.T) {
	clock = func() time.Time { return time.Time{} }
	if !clock().IsZero() {
		t.Fatal("clock was not replaced")
	}
}

// Accepted: the fake arrives through the constructor.
func TestRecorder(t *testing.T) {
	r := NewRecorder(func() time.Time { return time.Time{} })
	if !r.At().IsZero() {
		t.Fatal("recorder used the real clock")
	}
}
