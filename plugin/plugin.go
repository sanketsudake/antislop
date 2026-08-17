// Package plugin exposes antislop as a golangci-lint module plugin, so that
// `golangci-lint custom` can build a golangci-lint binary that runs every
// antislop analyzer under the single linter name "antislop".
package plugin

import (
	"encoding/json"
	"fmt"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/sanketsudake/antislop"
)

func init() { register.Plugin("antislop", New) }

// New is the golangci-lint module-plugin constructor. settings is the
// linters.settings.custom.antislop.settings block, already decoded from YAML
// into nil or a map; it is re-encoded as JSON and merged over the antislop
// defaults, so an option that is not mentioned keeps its default.
func New(settings any) (register.LinterPlugin, error) {
	raw, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("antislop: encode settings: %w", err)
	}
	cfg, err := antislop.ParseConfig(raw)
	if err != nil {
		return nil, err
	}
	return &linter{cfg: cfg}, nil
}

type linter struct {
	cfg antislop.Config
}

// BuildAnalyzers returns the analyzers enabled by the plugin settings.
func (p *linter) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return antislop.AnalyzersWith(p.cfg), nil
}

// GetLoadMode reports that antislop needs full type information.
func (p *linter) GetLoadMode() string { return register.LoadModeTypesInfo }
