package noanytypes_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/noanytypes"
)

func TestDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noanytypes.Analyzer, "a")
}
