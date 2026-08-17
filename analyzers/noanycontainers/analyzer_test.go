package noanycontainers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/noanycontainers"
)

func TestDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noanycontainers.Analyzer, "a", "encoders")
}

func TestNoEncoders(t *testing.T) {
	cfg := noanycontainers.Default()
	cfg.Encoders = nil
	analysistest.Run(t, analysistest.TestData(), noanycontainers.New(cfg), "noencoders")
}

func TestSlices(t *testing.T) {
	cfg := noanycontainers.Default()
	cfg.Slices = true
	analysistest.Run(t, analysistest.TestData(), noanycontainers.New(cfg), "withslices")
}

func TestSkipDeclaredAny(t *testing.T) {
	cfg := noanycontainers.Default()
	cfg.SkipDeclaredAny = true
	analysistest.Run(t, analysistest.TestData(), noanycontainers.New(cfg), "declared")
}
