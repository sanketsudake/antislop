package nochainedassert_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/nochainedassert"
)

func TestDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), nochainedassert.Analyzer, "a")
}
