package plugin_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/sanketsudake/antislop"
	"github.com/sanketsudake/antislop/plugin"
)

// The settings are written as JSON documents rather than map[string]any
// literals: golangci-lint hands New the YAML block as a map, and New marshals
// it straight back to JSON, so a raw document exercises the same code path.
// The map itself is covered end to end by scripts/plugin-smoke.sh.
func TestNewBuildsAnalyzers(t *testing.T) {
	all := antislop.Names()
	for _, tc := range []struct {
		name     string
		settings json.RawMessage
		want     []string
	}{
		{
			name:     "empty settings enable every analyzer",
			settings: json.RawMessage(`{}`),
			want:     all,
		},
		{
			name:     "null settings enable every analyzer",
			settings: json.RawMessage(`null`),
			want:     all,
		},
		{
			name:     "disable drops the named analyzer",
			settings: json.RawMessage(`{"disable":["noreflect"],"noanyparams":{"allow-variadic":false}}`),
			want:     without(all, "noreflect"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildNames(t, tc.settings); !slices.Equal(got, tc.want) {
				t.Errorf("analyzers = %v, want %v", got, tc.want)
			}
		})
	}
}

// An absent settings block reaches New as an untyped nil.
func TestNewNilSettings(t *testing.T) {
	p, err := plugin.New(nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	analyzers, err := p.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers: %v", err)
	}
	if len(analyzers) != len(antislop.Names()) {
		t.Errorf("built %d analyzers, want %d", len(analyzers), len(antislop.Names()))
	}
}

// Settings must reach the analyzer options, not only the flag defaults.
func TestNewAppliesOptions(t *testing.T) {
	p, err := plugin.New(json.RawMessage(`{"noanyparams":{"allow-variadic":false}}`))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	analyzers, err := p.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers: %v", err)
	}
	i := slices.IndexFunc(analyzers, func(a *analysis.Analyzer) bool { return a.Name == "noanyparams" })
	if i < 0 {
		t.Fatal("noanyparams not built")
	}
	flag := analyzers[i].Flags.Lookup("allow-variadic")
	if flag == nil {
		t.Fatal("noanyparams has no allow-variadic flag")
	}
	if got := flag.Value.String(); got != "false" {
		t.Errorf("allow-variadic = %s, want false", got)
	}
	// Options that are not mentioned keep their defaults.
	if got := analyzers[i].Flags.Lookup("allow-dictated").Value.String(); got != "true" {
		t.Errorf("allow-dictated = %s, want true", got)
	}
}

func TestNewRejectsInvalidSettings(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings json.RawMessage
	}{
		{name: "unknown analyzer in disable", settings: json.RawMessage(`{"disable":["nope"]}`)},
		{name: "ill-typed disable", settings: json.RawMessage(`{"disable":"noreflect"}`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := plugin.New(tc.settings); err == nil {
				t.Error("want an error, got nil")
			}
		})
	}
}

func TestLoadMode(t *testing.T) {
	p, err := plugin.New(nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := p.GetLoadMode(); got != register.LoadModeTypesInfo {
		t.Errorf("GetLoadMode = %q, want %q", got, register.LoadModeTypesInfo)
	}
}

// The plugin must be registered under the name golangci-lint configures.
func TestRegistered(t *testing.T) {
	if _, err := register.GetPlugin("antislop"); err != nil {
		t.Fatalf("GetPlugin: %v", err)
	}
}

func buildNames(t *testing.T, settings json.RawMessage) []string {
	t.Helper()
	p, err := plugin.New(settings)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	analyzers, err := p.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers: %v", err)
	}
	names := make([]string, 0, len(analyzers))
	for _, a := range analyzers {
		names = append(names, a.Name)
	}
	return names
}

func without(names []string, drop string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		if n != drop {
			out = append(out, n)
		}
	}
	return out
}
