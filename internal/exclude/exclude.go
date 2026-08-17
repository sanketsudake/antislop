// Package exclude drops diagnostics reported against paths the caller has
// exempted.
//
// It exists because the standalone binary is its own host: golangci-lint
// users scope a linter with issues.exclude-rules, but a repository running
// antislop directly has no such layer, and the alternative to a path
// exemption is disabling an analyzer for the whole module. That is a poor
// trade when one package legitimately uses a pattern the rule rejects — a
// domain term the nostructuralnames list happens to name, say — because it
// also stops the rule guarding every other package.
//
// Exclusion is a driver concern, not a rule concern: the analyzers stay
// free of any notion of which project is running them, and the same
// mechanism applies uniformly to every rule.
package exclude

import (
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/sanketsudake/antislop/internal/flagx"
)

// OptionName is the flag and settings key the exclusion option is bound
// under, on the driver-wide flag set and on every analyzer's.
const OptionName = "exclude"

// Global holds the patterns from the driver-wide -exclude flag, which apply
// to every analyzer. Each analyzer additionally has its own -<name>.exclude.
var Global []string

// Matches reports whether the file at filename is covered by any pattern.
//
// Patterns are matched against the path relative to dir (the directory the
// driver runs in, which for `antislop ./...` is the module root), written
// with forward slashes:
//
//   - "pkg/workflow/..." matches that directory and everything below it.
//   - "pkg/router/routeshape.go" matches exactly that file.
//   - a pattern containing no "/" matches against the base name alone, so
//     "*_test.go" covers test files at any depth.
//   - otherwise the pattern is matched with path.Match, whose "*" does not
//     cross a slash.
//
// A file outside dir is matched on its absolute path, so an unusual layout
// degrades to "no match" rather than to a surprising one.
func Matches(patterns []string, dir, filename string) bool {
	if len(patterns) == 0 {
		return false
	}
	name := relative(dir, filename)
	base := path.Base(name)
	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		if subtree, ok := strings.CutSuffix(pattern, "/..."); ok {
			if name == subtree || strings.HasPrefix(name, subtree+"/") {
				return true
			}
			continue
		}
		target := name
		if !strings.Contains(pattern, "/") {
			target = base
		}
		if ok, err := path.Match(pattern, target); err == nil && ok {
			return true
		}
	}
	return false
}

// relative renders filename relative to dir, with forward slashes. It falls
// back to the input when filename is not under dir.
func relative(dir, filename string) string {
	if dir == "" {
		return filepath.ToSlash(filename)
	}
	rel, err := filepath.Rel(dir, filename)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(filename)
	}
	return filepath.ToSlash(rel)
}

// Wrap binds an "exclude" option to a's flag set and filters the diagnostics
// a reports through it and through Global. It mutates a and returns it, and
// must be called at most once per analyzer.
//
// The filter replaces pass.Report rather than post-processing the driver's
// output, so it applies identically under multichecker, checker.Analyze, and
// any other driver.
func Wrap(a *analysis.Analyzer, dir func() string) *analysis.Analyzer {
	var own []string
	a.Flags.Var(flagx.NewList(&own), OptionName,
		"comma-separated path patterns whose findings this analyzer skips (\"pkg/gen/...\", \"*_test.go\")")
	inner := a.Run
	a.Run = func(pass *analysis.Pass) (any, error) {
		if len(own) > 0 || len(Global) > 0 {
			report := pass.Report
			pass.Report = func(d analysis.Diagnostic) {
				filename := pass.Fset.Position(d.Pos).Filename
				base := dir()
				if Matches(own, base, filename) || Matches(Global, base, filename) {
					return
				}
				report(d)
			}
		}
		return inner(pass)
	}
	return a
}
