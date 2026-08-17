package nowidenassert_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/nowidenassert"
)

func TestDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), nowidenassert.Analyzer, "a")
}
