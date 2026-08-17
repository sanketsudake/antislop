package antislop_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"

	"github.com/sanketsudake/antislop"
	"github.com/sanketsudake/antislop/internal/baseline"
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

// Every analyzer must carry the path-exclusion flag, bound once in the
// registry rather than by each rule — but it must NOT appear in Infos(),
// which feeds the golangci-lint settings the plugin path reads. That path
// scopes a linter with issues.exclude-rules and ignores this option, so
// listing it there would document something that does nothing.
func TestExcludeOptionIsBoundButNotAdvertisedToThePlugin(t *testing.T) {
	for _, a := range antislop.Analyzers() {
		if a.Flags.Lookup("exclude") == nil {
			t.Errorf("analyzer %q has no exclude flag", a.Name)
		}
	}
	for _, info := range antislop.Infos() {
		if slices.ContainsFunc(info.Options, func(o antislop.Option) bool { return o.Name == "exclude" }) {
			t.Errorf("analyzer %q advertises exclude to the plugin settings", info.Name)
		}
	}
}

// Exclusion has to filter what a driver actually reports, not merely match
// paths, so this runs the analyzers over the example module and checks that
// the findings for an excluded file disappear while the rest survive.
func TestExcludeFiltersReportedDiagnostics(t *testing.T) {
	const excluded = "noanyparams.go"

	all := summaryCounts(t)
	if all[excluded] == 0 {
		t.Fatalf("fixture problem: no findings in %s to exclude", excluded)
	}

	t.Run("per-analyzer exclude drops only that analyzer", func(t *testing.T) {
		got := summaryCounts(t, "-noanyparams.exclude", excluded)
		if got[excluded] >= all[excluded] {
			t.Errorf("findings in %s did not drop: %d -> %d", excluded, all[excluded], got[excluded])
		}
		for file, n := range all {
			if file != excluded && got[file] != n {
				t.Errorf("%s changed from %d to %d; exclusion leaked", file, n, got[file])
			}
		}
	})

	t.Run("driver-wide exclude drops the file entirely", func(t *testing.T) {
		got := summaryCounts(t, "-exclude", excluded)
		if got[excluded] != 0 {
			t.Errorf("%d finding(s) still reported in an excluded file", got[excluded])
		}
		for file, n := range all {
			if file != excluded && got[file] != n {
				t.Errorf("%s changed from %d to %d; exclusion leaked", file, n, got[file])
			}
		}
	})
}

// summaryCounts runs baseline -update over the example module and returns
// findings per file, which is the cheapest way to observe what a driver
// reported after filtering.
func summaryCounts(t *testing.T, extra ...string) map[string]int {
	t.Helper()
	path := filepath.Join(t.TempDir(), "baseline.txt")
	args := append([]string{"-baseline", path, "-update", "-dir=example"}, extra...)
	var out, errOut strings.Builder
	if code := baseline.Run(append(args, "./..."), antislop.Analyzers(), &out, &errOut); code != 0 {
		t.Fatalf("baseline -update exit %d: %s", code, errOut.String())
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, line := range strings.Split(string(body), "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		n, err := strconv.Atoi(fields[0])
		if err != nil {
			t.Fatalf("bad baseline line %q", line)
		}
		counts[fields[1]] += n
	}
	return counts
}
