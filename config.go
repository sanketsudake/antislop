package antislop

import (
	"encoding/json"
	"fmt"

	"github.com/sanketsudake/antislop/analyzers/noanycontainers"
	"github.com/sanketsudake/antislop/analyzers/noanyfields"
	"github.com/sanketsudake/antislop/analyzers/noanyparams"
	"github.com/sanketsudake/antislop/analyzers/noanyreturns"
	"github.com/sanketsudake/antislop/analyzers/noanytypes"
	"github.com/sanketsudake/antislop/analyzers/nochainedassert"
	"github.com/sanketsudake/antislop/analyzers/noknownwidening"
	"github.com/sanketsudake/antislop/analyzers/nomonkeypatch"
	"github.com/sanketsudake/antislop/analyzers/nonarrowany"
	"github.com/sanketsudake/antislop/analyzers/noreflect"
	"github.com/sanketsudake/antislop/analyzers/nostructuralnames"
	"github.com/sanketsudake/antislop/analyzers/nountypedunmarshal"
	"github.com/sanketsudake/antislop/analyzers/nowidenassert"
	"github.com/sanketsudake/antislop/analyzers/safetycomment"
)

// Config configures every antislop analyzer. It is the settings shape used by
// the golangci-lint module plugin; each analyzer also binds the same options
// as flags for the standalone binary.
type Config struct {
	// Disable lists analyzer names to leave out.
	Disable []string `json:"disable"`

	NoAnyParams        noanyparams.Config        `json:"noanyparams"`
	NoAnyReturns       noanyreturns.Config       `json:"noanyreturns"`
	NoAnyTypes         noanytypes.Config         `json:"noanytypes"`
	NoAnyFields        noanyfields.Config        `json:"noanyfields"`
	NoAnyContainers    noanycontainers.Config    `json:"noanycontainers"`
	NoNarrowAny        nonarrowany.Config        `json:"nonarrowany"`
	SafetyComment      safetycomment.Config      `json:"safetycomment"`
	NoChainedAssert    nochainedassert.Config    `json:"nochainedassert"`
	NoKnownWidening    noknownwidening.Config    `json:"noknownwidening"`
	NoWidenAssert      nowidenassert.Config      `json:"nowidenassert"`
	NoReflect          noreflect.Config          `json:"noreflect"`
	NoMonkeyPatch      nomonkeypatch.Config      `json:"nomonkeypatch"`
	NoUntypedUnmarshal nountypedunmarshal.Config `json:"nountypedunmarshal"`
	NoStructuralNames  nostructuralnames.Config  `json:"nostructuralnames"`
}

// DefaultConfig returns every analyzer's default options.
func DefaultConfig() Config {
	return Config{
		NoAnyParams:        noanyparams.Default(),
		NoAnyReturns:       noanyreturns.Default(),
		NoAnyTypes:         noanytypes.Default(),
		NoAnyFields:        noanyfields.Default(),
		NoAnyContainers:    noanycontainers.Default(),
		NoNarrowAny:        nonarrowany.Default(),
		SafetyComment:      safetycomment.Default(),
		NoChainedAssert:    nochainedassert.Default(),
		NoKnownWidening:    noknownwidening.Default(),
		NoWidenAssert:      nowidenassert.Default(),
		NoReflect:          noreflect.Default(),
		NoMonkeyPatch:      nomonkeypatch.Default(),
		NoUntypedUnmarshal: nountypedunmarshal.Default(),
		NoStructuralNames:  nostructuralnames.Default(),
	}
}

// ParseConfig decodes a JSON document on top of DefaultConfig, so options
// that are not mentioned keep their defaults. Empty input yields the defaults.
func ParseConfig(raw []byte) (Config, error) {
	cfg := DefaultConfig()
	if len(raw) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("antislop: decode settings: %w", err)
	}
	for _, name := range cfg.Disable {
		if _, ok := byName[name]; !ok {
			return cfg, fmt.Errorf("antislop: unknown analyzer %q in disable", name)
		}
	}
	return cfg, nil
}
