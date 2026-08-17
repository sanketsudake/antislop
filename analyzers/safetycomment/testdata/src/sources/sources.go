package sources

import (
	"bytes"
	"context"
	"os"
	"sync"
	"syscall"
)

type key struct{}

// An unchecked assertion on the immediate result of a standard-library API
// that returns an untyped value is not reported by default.

func fromContext(ctx context.Context) string {
	return ctx.Value(key{}).(string)
}

var pool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func fromPool() *bytes.Buffer {
	return pool.Get().(*bytes.Buffer)
}

func fromSyncMap(m *sync.Map, k string) int {
	if v, ok := m.Load(k); ok {
		return v.(int)
	}
	return 0
}

// A value that did not come from a listed source still needs SAFETY.

var raw any

func fromRaw() int {
	return raw.(int) // want `safetycomment: unchecked type assertion has no SAFETY: justification`
}

// os.FileInfo.Sys returns any by design.
func statMode(fi os.FileInfo) *syscall.Stat_t { return fi.Sys().(*syscall.Stat_t) }
