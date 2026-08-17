package stubbing

import (
	"testing"
	"time"
)

// Accepted because allow-package-var-stubbing is set: the import is still
// reported, only the hand-written stub is allowed.
func TestPatchedClock(t *testing.T) {
	timeNow = func() time.Time { return time.Time{} }
	_ = timeNow()
}
