package exclude_test

import (
	"path/filepath"
	"testing"

	"github.com/sanketsudake/antislop/internal/exclude"
)

func TestMatches(t *testing.T) {
	root := filepath.FromSlash("/repo")
	for _, tc := range []struct {
		name     string
		patterns []string
		file     string
		want     bool
	}{
		{name: "no patterns matches nothing", file: "/repo/pkg/a/a.go"},
		{name: "empty pattern is ignored", patterns: []string{""}, file: "/repo/pkg/a/a.go"},

		{name: "subtree covers a nested file", patterns: []string{"pkg/gen/..."}, file: "/repo/pkg/gen/deep/x.go", want: true},
		{name: "subtree covers a direct child", patterns: []string{"pkg/gen/..."}, file: "/repo/pkg/gen/x.go", want: true},
		{name: "subtree does not cover a sibling", patterns: []string{"pkg/gen/..."}, file: "/repo/pkg/genuine/x.go"},
		{name: "subtree does not cover a parent", patterns: []string{"pkg/gen/..."}, file: "/repo/pkg/x.go"},

		{name: "exact path", patterns: []string{"pkg/a/a.go"}, file: "/repo/pkg/a/a.go", want: true},
		{name: "exact path does not match elsewhere", patterns: []string{"pkg/a/a.go"}, file: "/repo/pkg/b/a.go"},

		{name: "bare pattern matches the base name at any depth", patterns: []string{"*_test.go"}, file: "/repo/pkg/a/b/x_test.go", want: true},
		{name: "bare pattern ignores non-matching base", patterns: []string{"*_test.go"}, file: "/repo/pkg/a/x.go"},

		{name: "star does not cross a slash", patterns: []string{"pkg/*.go"}, file: "/repo/pkg/a/x.go"},
		{name: "star within one segment", patterns: []string{"pkg/*.go"}, file: "/repo/pkg/x.go", want: true},

		{name: "any pattern in the list can match", patterns: []string{"nope/...", "pkg/a/..."}, file: "/repo/pkg/a/x.go", want: true},

		// A file outside the run directory keeps its absolute path, so a
		// relative pattern must not accidentally cover it.
		{name: "file outside root", patterns: []string{"pkg/a/..."}, file: "/elsewhere/pkg/a/x.go"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := exclude.Matches(tc.patterns, root, filepath.FromSlash(tc.file))
			if got != tc.want {
				t.Errorf("Matches(%q, %q) = %v, want %v", tc.patterns, tc.file, got, tc.want)
			}
		})
	}
}

// An empty run directory falls back to the absolute path rather than
// panicking or matching everything.
func TestMatchesWithoutRoot(t *testing.T) {
	if exclude.Matches([]string{"pkg/a/..."}, "", filepath.FromSlash("/repo/pkg/a/x.go")) {
		t.Error("a relative pattern must not match an absolute path")
	}
	if !exclude.Matches([]string{"*.go"}, "", filepath.FromSlash("/repo/pkg/a/x.go")) {
		t.Error("a base-name pattern should still match")
	}
}
