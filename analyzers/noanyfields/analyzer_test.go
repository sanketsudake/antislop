package noanyfields_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/noanyfields"
)

func TestDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noanyfields.Analyzer, "a")
}
