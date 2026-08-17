package nonarrowany_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/nonarrowany"
)

func TestDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), nonarrowany.Analyzer, "a", "sources", "foreign")
}

func TestNoSources(t *testing.T) {
	cfg := nonarrowany.Default()
	cfg.Sources = nil
	analysistest.Run(t, analysistest.TestData(), nonarrowany.New(cfg), "nosources")
}

func TestAllowInParseFuncs(t *testing.T) {
	cfg := nonarrowany.Default()
	cfg.AllowInParseFuncs = true
	analysistest.Run(t, analysistest.TestData(), nonarrowany.New(cfg), "parsefuncs")
}

func TestSkipDeclaredAny(t *testing.T) {
	cfg := nonarrowany.Default()
	cfg.SkipDeclaredAny = true
	analysistest.Run(t, analysistest.TestData(), nonarrowany.New(cfg), "anyparams")
}
