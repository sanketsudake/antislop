package baseline_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sanketsudake/antislop"
	"github.com/sanketsudake/antislop/internal/baseline"
)

func TestWanted(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{args: []string{"-baseline", "b.txt", "./..."}, want: true},
		{args: []string{"--baseline", "b.txt"}, want: true},
		{args: []string{"-baseline=b.txt", "./..."}, want: true},
		{args: []string{"--baseline=b.txt"}, want: true},
		{args: []string{"./..."}},
		{args: []string{"-summary", "./..."}},
		{args: []string{"--", "-baseline", "b.txt"}},
		// -baseline with no value is not a request for the mode; the
		// multichecker path reports the bad flag.
		{args: []string{"-baseline"}},
	} {
		if got := baseline.Wanted(tc.args); got != tc.want {
			t.Errorf("Wanted(%q) = %v, want %v", tc.args, got, tc.want)
		}
	}
}

// The example module is the golden fixture, as it is for summary: -update
// must record exactly the findings example/expected.txt lists, and gating
// against what it wrote must then pass.
func TestUpdateThenGate(t *testing.T) {
	golden, err := os.ReadFile("../../example/expected.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := len(strings.Split(strings.TrimSpace(string(golden)), "\n"))
	path := filepath.Join(t.TempDir(), "baseline.txt")

	var out, errOut bytes.Buffer
	if code := run(t, &out, &errOut, "-baseline", path, "-update", "-dir=../../example", "./..."); code != 0 {
		t.Fatalf("-update exit %d, stderr: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), strconv.Itoa(want)+" accepted finding") {
		t.Fatalf("-update did not record %d findings:\n%s", want, out.String())
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(body), "#") {
		t.Error("the written baseline should start with its explanatory header")
	}
	// Paths are recorded relative to the run directory, so the file is
	// portable across checkouts.
	if strings.Contains(string(body), t.TempDir()) || strings.Contains(string(body), "/Users/") {
		t.Errorf("baseline records absolute paths:\n%s", body)
	}

	out.Reset()
	errOut.Reset()
	if code := run(t, &out, &errOut, "-baseline", path, "-dir=../../example", "./..."); code != 0 {
		t.Fatalf("gate against a fresh baseline exit %d:\n%s\n%s", code, out.String(), errOut.String())
	}
}

// A pair the baseline does not list fails, and the report names the actual
// diagnostics rather than only the arithmetic.
func TestGateFailsOnUnlistedPair(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.txt")
	if err := os.WriteFile(path, []byte("# empty\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run(t, &out, &errOut, "-baseline", path, "-dir=../../example", "./..."); code != 3 {
		t.Fatalf("exit %d, want 3\n%s\n%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "baseline allows 0") {
		t.Errorf("report should name the allowance:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "has type any") {
		t.Errorf("report should include the diagnostics themselves:\n%s", out.String())
	}
}

// A count below the baseline passes: the file is a ceiling, not an equality.
func TestGateAllowsShrink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.txt")
	var out, errOut bytes.Buffer
	if code := run(t, &out, &errOut, "-baseline", path, "-update", "-dir=../../example", "./..."); code != 0 {
		t.Fatalf("-update exit %d", code)
	}
	out.Reset()
	errOut.Reset()
	code := run(t, &out, &errOut, "-baseline", path, "-dir=../../example", "-noanyparams=false", "./...")
	if code != 0 {
		t.Fatalf("a shrinking count must pass, got exit %d:\n%s", code, out.String())
	}
}

func TestMalformedBaselineIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.txt")
	if err := os.WriteFile(path, []byte("not a count line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run(t, &out, &errOut, "-baseline", path, "-dir=../../example", "./..."); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "<count> <file> <analyzer>") {
		t.Errorf("the error should show the expected shape: %s", errOut.String())
	}
}

func TestMissingPathIsAnError(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run(t, &out, &errOut, "-baseline=", "-dir=../../example", "./..."); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
}

// run invokes baseline mode with a fresh analyzer set, so flag state set by
// one case cannot leak into the next.
func run(t *testing.T, out, errOut *bytes.Buffer, args ...string) int {
	t.Helper()
	return baseline.Run(args, antislop.Analyzers(), out, errOut)
}
