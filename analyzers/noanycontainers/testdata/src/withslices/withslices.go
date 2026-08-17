package withslices

// --- invalid with slices: true -------------------------------------------------

var anySlice []any // want `noanycontainers: \[\]any erases the element types`

var anyArray [3]any // want `noanycontainers: \[3\]any erases the element types`

var anyChan chan any // want `noanycontainers: chan any erases the element types`

var recvChan <-chan any // want `noanycontainers: <-chan any erases the element types`

var sendChan chan<- any // want `noanycontainers: chan<- any erases the element types`

// The map is not the offender: its direct value type is a slice.
var nested map[string][]any // want `noanycontainers: \[\]any erases the element types`

// --- valid ---------------------------------------------------------------------

var ints []int

func generic[T any](s []T) []T { return s }

func variadic(args ...any) { _ = args }
