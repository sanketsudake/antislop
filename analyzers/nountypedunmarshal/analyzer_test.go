package nountypedunmarshal_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/nountypedunmarshal"
)

func TestDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), nountypedunmarshal.Analyzer, "a")
}

func TestFunctionsReplacesTheList(t *testing.T) {
	cfg := nountypedunmarshal.Default()
	cfg.Functions = []string{"decoder.Decode#1"}
	analysistest.Run(t, analysistest.TestData(), nountypedunmarshal.New(cfg), "functions")
}

// A malformed entry must not panic at construction: the analyzer reports it as
// an error on its first run, before it looks at the package.
func TestMalformedFunctionSpec(t *testing.T) {
	for _, entry := range []string{"nonsense", "pkg.Func", "pkg.Func#x", "(*pkg.Type.Method#0", "(*pkg.Type)Method#0", ".Func#0"} {
		a := nountypedunmarshal.New(nountypedunmarshal.Config{Functions: []string{entry}})
		if _, err := a.Run(nil); err == nil {
			t.Errorf("entry %q: want an error, got none", entry)
		}
	}
}
