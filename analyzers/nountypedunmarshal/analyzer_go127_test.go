//go:build go1.27

package nountypedunmarshal_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/nountypedunmarshal"
)

// TestGo127 covers the encoding/json/v2 and encoding/json/jsontext targets.
// It is built only by a Go 1.27 toolchain: before that release those packages
// were reachable only under GOEXPERIMENT=jsonv2.
func TestGo127(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), nountypedunmarshal.Analyzer, "go127")
}
