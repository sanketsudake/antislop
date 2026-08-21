package langver

import "testing"

// TestAtLeast pins the version decision the analysistest fixtures cannot
// reach: they run without a go.mod, so the package version is always empty
// there and only the file version varies.
func TestAtLeast(t *testing.T) {
	for _, tc := range []struct {
		name      string
		file, pkg string
		want      string
		expect    bool
	}{
		{"file at want", "go1.27", "", go127, true},
		{"file past want", "go1.28", "", go127, true},
		{"file below want", "go1.26", "", go127, false},
		{"file wins over older package", "go1.27", "go1.26.0", go127, true},
		{"file wins over newer package", "go1.26", "go1.28", go127, false},
		{"package used when file is unset", "", "go1.27", go127, true},
		{"package below want", "", "go1.26.0", go127, false},
		{"patch version compares as its release", "", "go1.27.3", go127, true},
		{"neither stated reports false", "", "", go127, false},
		{"malformed version reports false", "1.27", "", go127, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := atLeast(tc.file, tc.pkg, tc.want); got != tc.expect {
				t.Errorf("atLeast(%q, %q, %q) = %v, want %v", tc.file, tc.pkg, tc.want, got, tc.expect)
			}
		})
	}
}
