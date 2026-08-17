package safetycomment_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/safetycomment"
)

func TestDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), safetycomment.Analyzer, "a", "sources", "foreign")
}

func TestNoSources(t *testing.T) {
	cfg := safetycomment.Default()
	cfg.Sources = nil
	analysistest.Run(t, analysistest.TestData(), safetycomment.New(cfg), "nosources")
}

func TestSkipDeclaredAny(t *testing.T) {
	cfg := safetycomment.Default()
	cfg.SkipDeclaredAny = true
	analysistest.Run(t, analysistest.TestData(), safetycomment.New(cfg), "anyparams")
}
