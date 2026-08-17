// Package antislop registers opinionated go/analysis analyzers that reject
// low-evidence Go patterns: empty-interface escape hatches, unchecked
// narrowing, reflection, monkey patching, structural names, and untyped
// decoding.
package antislop

import (
	"flag"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"

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
	"github.com/sanketsudake/antislop/internal/flagx"
)

type entry struct {
	name    string
	def     *analysis.Analyzer
	build   func(Config) *analysis.Analyzer
	summary string
}

// registry is the ordered list of analyzers (the src/index.ts analogue).
var registry = []entry{
	{
		name:    noanyparams.Name,
		def:     noanyparams.Analyzer,
		build:   func(c Config) *analysis.Analyzer { return noanyparams.New(c.NoAnyParams) },
		summary: "parameters typed as the empty interface",
	},
	{
		name:    noanyreturns.Name,
		def:     noanyreturns.Analyzer,
		build:   func(c Config) *analysis.Analyzer { return noanyreturns.New(c.NoAnyReturns) },
		summary: "function results typed as the empty interface",
	},
	{
		name:    noanytypes.Name,
		def:     noanytypes.Analyzer,
		build:   func(c Config) *analysis.Analyzer { return noanytypes.New(c.NoAnyTypes) },
		summary: "type declarations that merely rename the empty interface",
	},
	{
		name:    noanyfields.Name,
		def:     noanyfields.Analyzer,
		build:   func(c Config) *analysis.Analyzer { return noanyfields.New(c.NoAnyFields) },
		summary: "struct fields typed as the empty interface",
	},
	{
		name:    noanycontainers.Name,
		def:     noanycontainers.Analyzer,
		build:   func(c Config) *analysis.Analyzer { return noanycontainers.New(c.NoAnyContainers) },
		summary: "maps (and optionally slices, arrays and channels) whose element or key type is the empty interface",
	},
	{
		name:    nonarrowany.Name,
		def:     nonarrowany.Analyzer,
		build:   func(c Config) *analysis.Analyzer { return nonarrowany.New(c.NoNarrowAny) },
		summary: "checked narrowing of an empty-interface value (comma-ok assertions and type switches)",
	},
	{
		name:    safetycomment.Name,
		def:     safetycomment.Analyzer,
		build:   func(c Config) *analysis.Analyzer { return safetycomment.New(c.SafetyComment) },
		summary: "unchecked escape hatches with no SAFETY comment: single-value type assertions, unsafe conversions, and go:linkname",
	},
	{
		name:    nochainedassert.Name,
		def:     nochainedassert.Analyzer,
		build:   func(c Config) *analysis.Analyzer { return nochainedassert.New(c.NoChainedAssert) },
		summary: "assertions chained through the empty interface (x.(any).(T), any(x).(T))",
	},
	{
		name:    noknownwidening.Name,
		def:     noknownwidening.Analyzer,
		build:   func(c Config) *analysis.Analyzer { return noknownwidening.New(c.NoKnownWidening) },
		summary: "known concrete values stored into empty-interface locations",
	},
	{
		name:    nowidenassert.Name,
		def:     nowidenassert.Analyzer,
		build:   func(c Config) *analysis.Analyzer { return nowidenassert.New(c.NoWidenAssert) },
		summary: "values widened to the empty interface and later asserted back in the same function",
	},
	{
		name:    noreflect.Name,
		def:     noreflect.Analyzer,
		build:   func(c Config) *analysis.Analyzer { return noreflect.New(c.NoReflect) },
		summary: "dynamic access and invocation through package reflect",
	},
	{
		name:    nomonkeypatch.Name,
		def:     nomonkeypatch.Analyzer,
		build:   func(c Config) *analysis.Analyzer { return nomonkeypatch.New(c.NoMonkeyPatch) },
		summary: "monkey patching: patch libraries and test-time reassignment of package-level function variables",
	},
	{
		name:    nountypedunmarshal.Name,
		def:     nountypedunmarshal.Analyzer,
		build:   func(c Config) *analysis.Analyzer { return nountypedunmarshal.New(c.NoUntypedUnmarshal) },
		summary: "decoding into untyped targets (any, map[string]any, []any)",
	},
	{
		name:    nostructuralnames.Name,
		def:     nostructuralnames.Analyzer,
		build:   func(c Config) *analysis.Analyzer { return nostructuralnames.New(c.NoStructuralNames) },
		summary: "identifiers that contain a forbidden structural term (default: shape)",
	},
}

var byName = func() map[string]entry {
	m := make(map[string]entry, len(registry))
	for _, e := range registry {
		m[e.name] = e
	}
	return m
}()

// Analyzers returns every analyzer with default options and flag bindings,
// for the standalone multichecker driver.
func Analyzers() []*analysis.Analyzer {
	out := make([]*analysis.Analyzer, 0, len(registry))
	for _, e := range registry {
		out = append(out, e.def)
	}
	return out
}

// AnalyzersWith builds every enabled analyzer from cfg.
func AnalyzersWith(cfg Config) []*analysis.Analyzer {
	disabled := make(map[string]bool, len(cfg.Disable))
	for _, n := range cfg.Disable {
		disabled[n] = true
	}
	out := make([]*analysis.Analyzer, 0, len(registry))
	for _, e := range registry {
		if !disabled[e.name] {
			out = append(out, e.build(cfg))
		}
	}
	return out
}

// Names returns the analyzer names in registry order.
func Names() []string {
	out := make([]string, 0, len(registry))
	for _, e := range registry {
		out = append(out, e.name)
	}
	return out
}

// Option describes one analyzer option, as bound to the analyzer's flag set.
type Option struct {
	// Name is the option name, shared by the command-line flag and the
	// host's settings key.
	Name string
	// Default is the default value written the way a settings file spells
	// it: the flag's default string for scalar options, and a quoted flow
	// sequence for list options, whose settings value is an array rather
	// than the comma-separated string the command line accepts.
	Default string
	// Usage is the one-line flag usage text.
	Usage string
}

// Info describes a registered analyzer for documentation generators.
type Info struct {
	Name    string
	Summary string
	Doc     string
	Options []Option
}

// Infos returns documentation info for every analyzer in registry order.
func Infos() []Info {
	out := make([]Info, 0, len(registry))
	for _, e := range registry {
		out = append(out, Info{
			Name:    e.name,
			Summary: e.summary,
			Doc:     e.def.Doc,
			Options: options(e.def),
		})
	}
	return out
}

// options lists an analyzer's options in flag-set order, which is
// alphabetical and therefore stable across builds.
func options(a *analysis.Analyzer) []Option {
	var out []Option
	a.Flags.VisitAll(func(f *flag.Flag) {
		out = append(out, Option{
			Name:    f.Name,
			Default: defaultLiteral(f),
			Usage:   strings.Join(strings.Fields(f.Usage), " "),
		})
	})
	return out
}

// defaultLiteral renders f's default value as a settings literal. A list
// option reaches the analyzer as a JSON array, so its comma-separated flag
// default is rewritten as a flow sequence of quoted entries.
func defaultLiteral(f *flag.Flag) string {
	if _, isList := f.Value.(*flagx.List); !isList {
		return f.DefValue
	}
	if f.DefValue == "" {
		return "[]"
	}
	entries := strings.Split(f.DefValue, ",")
	for i, entry := range entries {
		entries[i] = strconv.Quote(entry)
	}
	return "[" + strings.Join(entries, ", ") + "]"
}
