package noanyreturns_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/noanyreturns"
)

func TestDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noanyreturns.Analyzer, "a")
}

func TestStrict(t *testing.T) {
	cfg := noanyreturns.Default()
	cfg.AllowDictated = false
	analysistest.Run(t, analysistest.TestData(), noanyreturns.New(cfg), "strict")
}
