package nomonkeypatch_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sanketsudake/antislop/analyzers/nomonkeypatch"
)

func TestDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), nomonkeypatch.Analyzer, "a")
}

func TestAllowPackageVarStubbing(t *testing.T) {
	cfg := nomonkeypatch.Default()
	cfg.AllowPackageVarStubbing = true
	analysistest.Run(t, analysistest.TestData(), nomonkeypatch.New(cfg), "stubbing")
}

func TestPackagesReplacesTheList(t *testing.T) {
	cfg := nomonkeypatch.Default()
	cfg.Packages = []string{"bou.ke/monkey"}
	analysistest.Run(t, analysistest.TestData(), nomonkeypatch.New(cfg), "packages")
}
