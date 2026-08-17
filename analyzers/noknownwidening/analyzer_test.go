package noknownwidening_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/noknownwidening"
)

func TestDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noknownwidening.Analyzer, "a")
}

func TestStrict(t *testing.T) {
	cfg := noknownwidening.Default()
	cfg.AllowVariadicArgs = false
	analysistest.Run(t, analysistest.TestData(), noknownwidening.New(cfg), "strict")
}
