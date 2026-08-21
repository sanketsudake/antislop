//go:build go1.27

package noanyreturns_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/noanyreturns"
)

// TestGo127 covers the generic-method advice. It is built only by a Go 1.27
// toolchain: the fixture declares generic methods, which a Go 1.26 toolchain
// cannot compile, and analysistest type-checks fixtures with the toolchain
// running the test.
func TestGo127(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noanyreturns.Analyzer, "go127")
}
