package nostructuralnames_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/nostructuralnames"
)

func TestDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), nostructuralnames.Analyzer, "a")
}

func TestPackageClause(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), nostructuralnames.Analyzer, "shapes")
}

func TestTermsReplacesTheList(t *testing.T) {
	cfg := nostructuralnames.Default()
	cfg.Terms = []string{"widget", "gizmo"}
	analysistest.Run(t, analysistest.TestData(), nostructuralnames.New(cfg), "terms")
}
