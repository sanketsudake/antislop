package noanyparams_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/noanyparams"
)

func TestDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noanyparams.Analyzer, "a")
}

func TestStrict(t *testing.T) {
	cfg := noanyparams.Default()
	cfg.AllowVariadic = false
	cfg.AllowDictated = false
	analysistest.Run(t, analysistest.TestData(), noanyparams.New(cfg), "strict")
}
