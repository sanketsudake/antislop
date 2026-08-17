// Package findings turns an analysis graph into the deduplicated list of
// diagnostics the driver modes report on.
//
// Deduplication is the reason this is shared rather than written per mode:
// loading with Tests: true analyzes a package and its test variant, so the
// same diagnostic is reported more than once. A baseline written by one mode
// and checked by another has to agree on the count, or the gate flaps.
package findings

import (
	"strings"

	"golang.org/x/tools/go/analysis/checker"
)

// Finding is one diagnostic, counted once.
type Finding struct {
	// Analyzer is the rule that reported it.
	Analyzer string
	// File is the absolute path of the file it was reported against.
	File string
	// Line is the 1-based line number.
	Line int
	// Package is the import path, with the test-variant suffixes trimmed so
	// a package and its tests aggregate together.
	Package string
	// Message is the diagnostic text.
	Message string
	// InTest reports whether File is a _test.go file.
	InTest bool
}

// Collect walks the root actions of graph and returns every diagnostic once,
// in the order encountered.
func Collect(graph *checker.Graph) []Finding {
	var out []Finding
	seen := map[string]bool{}
	for act := range graph.All() {
		if !act.IsRoot {
			continue
		}
		fset := act.Package.Fset
		for _, d := range act.Diagnostics {
			pos := fset.Position(d.Pos)
			key := act.Analyzer.Name + "\x00" + pos.String() + "\x00" + d.Message
			if seen[key] {
				continue
			}
			seen[key] = true
			pkgPath := strings.TrimSuffix(act.Package.PkgPath, "_test")
			pkgPath = strings.TrimSuffix(pkgPath, ".test")
			out = append(out, Finding{
				Analyzer: act.Analyzer.Name,
				File:     pos.Filename,
				Line:     pos.Line,
				Package:  pkgPath,
				Message:  d.Message,
				InTest:   strings.HasSuffix(pos.Filename, "_test.go"),
			})
		}
	}
	return out
}
