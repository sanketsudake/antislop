package example

// Rejected by nowidenassert: boxed is widened and asked about in the same
// breath. The declaration also trips noknownwidening, and the comma-ok form
// trips nonarrowany.
func Double(n int) int {
	var boxed any = n
	if v, ok := boxed.(int); ok {
		return v * 2
	}
	return 0
}

// Accepted: the type the function knew is the type it uses.
func DoubleDirect(n int) int { return n * 2 }
