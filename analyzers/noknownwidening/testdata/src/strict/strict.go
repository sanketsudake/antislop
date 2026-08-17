package strict

import "fmt"

// --- invalid: with allow-variadic-args off, forwarding is widening too -------

func forwarding() {
	fmt.Println(any(42)) // want `noknownwidening: value of type int is stored as any`
	_ = []any{any(1)}    // want `noknownwidening: value of type int is stored as any`
}

// --- valid -------------------------------------------------------------------

// The exemption only ever covered explicit conversions; a bare argument is
// still noanyparams' report.
func bare() {
	fmt.Println(42)
}
