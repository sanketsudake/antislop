package noreflect_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/noreflect"
)

func TestDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noreflect.Analyzer, "a")
}

func TestStrict(t *testing.T) {
	cfg := noreflect.Default()
	cfg.Strict = true
	analysistest.Run(t, analysistest.TestData(), noreflect.New(cfg), "strict")
}
