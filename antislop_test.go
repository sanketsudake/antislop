package antislop_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"

	"github.com/sanketsudake/antislop"
)

func TestValidate(t *testing.T) {
	if err := analysis.Validate(antislop.Analyzers()); err != nil {
		t.Fatal(err)
	}
	if err := analysis.Validate(antislop.AnalyzersWith(antislop.DefaultConfig())); err != nil {
		t.Fatal(err)
	}
}

// Every package under analyzers/ must be registered.
func TestEveryAnalyzerRegistered(t *testing.T) {
	entries, err := os.ReadDir("analyzers")
	if err != nil {
		t.Fatal(err)
	}
	names := antislop.Names()
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !slices.Contains(names, e.Name()) {
			t.Errorf("analyzers/%s is not registered in antislop.go", e.Name())
		}
		if _, err := os.Stat(filepath.Join("analyzers", e.Name(), "testdata")); err != nil {
			t.Errorf("analyzers/%s has no testdata fixtures", e.Name())
		}
	}
	if len(names) != len(slices.Compact(slices.Sorted(slices.Values(names)))) {
		t.Errorf("duplicate analyzer names: %v", names)
	}
}

func TestParseConfig(t *testing.T) {
	cfg, err := antislop.ParseConfig([]byte(`{"disable":["noanyparams"],"noanyparams":{"allow-variadic":false}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NoAnyParams.AllowVariadic || !cfg.NoAnyParams.AllowDictated {
		t.Errorf("options not merged over defaults: %+v", cfg.NoAnyParams)
	}
	if got := antislop.AnalyzersWith(cfg); len(got) != len(antislop.Names())-1 {
		t.Errorf("disable not honoured: got %d analyzers", len(got))
	}
	if _, err := antislop.ParseConfig([]byte(`{"disable":["nope"]}`)); err == nil {
		t.Error("unknown analyzer in disable should error")
	}
	if _, err := antislop.ParseConfig(nil); err != nil {
		t.Errorf("empty settings: %v", err)
	}
}

// Every option default rendered by Infos must be a literal the settings
// decoder accepts, and must decode back to the analyzer's own default. This
// is what makes the generated skills/install-antislop/assets/golangci-settings.yml
// safe to paste into a .golangci.yml.
func TestInfoOptionDefaultsRoundTrip(t *testing.T) {
	var doc strings.Builder
	doc.WriteString("{")
	for _, info := range antislop.Infos() {
		if len(info.Options) == 0 {
			continue
		}
		fmt.Fprintf(&doc, "%q:{", info.Name)
		for i, opt := range info.Options {
			if i > 0 {
				doc.WriteString(",")
			}
			fmt.Fprintf(&doc, "%q:%s", opt.Name, opt.Default)
		}
		doc.WriteString("},")
	}
	doc.WriteString(`"disable":[]}`)

	cfg, err := antislop.ParseConfig([]byte(doc.String()))
	if err != nil {
		t.Fatalf("defaults are not a valid settings document: %v\n%s", err, doc.String())
	}
	want := antislop.DefaultConfig()
	want.Disable = []string{}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("rendered defaults decode to %+v, want %+v", cfg, want)
	}
}
