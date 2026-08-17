// Package baseline implements the standalone binary's -baseline mode: it
// gates a tree against a recorded set of accepted findings instead of
// requiring the tree to be clean.
//
// It is a ratchet, not an amnesty. The file records a count per file and
// analyzer; a pair that is absent, or that grows, fails. A pair that shrinks
// passes and can be re-recorded. That lets a repository adopt antislop on an
// existing codebase without either disabling rules module-wide or blocking
// on a full cleanup, while still failing every new finding.
//
// The file deliberately carries no line numbers. Keying on them would churn
// the baseline on every edit above a finding, which is the failure mode that
// makes teams stop regenerating it and start ignoring it. The cost is that a
// net-zero swap inside one file and analyzer — remove one finding, add
// another — is not caught; diff stability is worth more.
package baseline

import (
	"bufio"
	"cmp"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"

	"github.com/sanketsudake/antislop/internal/exclude"
	"github.com/sanketsudake/antislop/internal/findings"
	"github.com/sanketsudake/antislop/internal/flagx"
)

// header is written above a generated baseline. Everything from a '#' to the
// end of the line is ignored when the file is read back.
const header = `# antislop findings accepted for this tree, written by 'antislop -baseline <file> -update'.
#
# Format: <count> <file> <analyzer>. No line numbers on purpose, so an edit
# above a finding does not churn this file.
#
# The gate fails when a file/analyzer pair is missing from this list or its
# count grows; a pair that shrinks passes. Every line is a claim that the
# pattern is right for that file — justify new ones in review.
`

// Wanted reports whether args request baseline mode, in either the
// -baseline=<path> or the -baseline <path> spelling.
func Wanted(args []string) bool {
	for i, a := range args {
		if a == "--" {
			return false
		}
		if a == "-baseline" || a == "--baseline" {
			return i+1 < len(args)
		}
		if strings.HasPrefix(a, "-baseline=") || strings.HasPrefix(a, "--baseline=") {
			return true
		}
	}
	return false
}

// entry is one recorded line: how many findings an analyzer may report
// against a file.
type entry struct {
	count    int
	file     string
	analyzer string
}

// key identifies the pair an entry counts.
type key struct {
	file     string
	analyzer string
}

// Run executes baseline mode and returns the process exit code: 0 when the
// tree is within the baseline, 3 when it is not, 1 on error.
func Run(args []string, analyzers []*analysis.Analyzer, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("antislop -baseline", flag.ContinueOnError)
	fs.SetOutput(stderr)
	path := fs.String("baseline", "", "path to the accepted-findings file")
	update := fs.Bool("update", false, "rewrite the baseline from the current tree instead of gating")
	tests := fs.Bool("test", true, "analyze test files too")
	dir := fs.String("dir", "", "directory to run in (default: current)")
	fs.Var(flagx.NewList(&exclude.Global), "exclude",
		"comma-separated path patterns whose findings every analyzer skips")
	enabled := map[string]*bool{}
	for _, a := range analyzers {
		enabled[a.Name] = fs.Bool(a.Name, true, "enable the "+a.Name+" analyzer")
		a.Flags.VisitAll(func(f *flag.Flag) {
			fs.Var(f.Value, a.Name+"."+f.Name, f.Usage)
		})
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *path == "" {
		return fail(stderr, fmt.Errorf("-baseline needs a file path"))
	}

	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	var selected []*analysis.Analyzer
	for _, a := range analyzers {
		if *enabled[a.Name] {
			selected = append(selected, a)
		}
	}

	root := *dir
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fail(stderr, err)
		}
		root = wd
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return fail(stderr, err)
	}
	root = abs

	pkgs, err := packages.Load(&packages.Config{
		Mode:  packages.LoadAllSyntax | packages.NeedModule,
		Tests: *tests,
		Dir:   *dir,
	}, patterns...)
	if err != nil {
		return fail(stderr, err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		return 1
	}
	graph, err := checker.Analyze(selected, pkgs, nil)
	if err != nil {
		return fail(stderr, err)
	}

	current := findings.Collect(graph)
	counts := tally(current, root)

	if *update {
		if err := write(*path, counts); err != nil {
			return fail(stderr, err)
		}
		if _, err := fmt.Fprintf(stdout, "antislop: wrote %d accepted finding(s) to %s\n", total(counts), *path); err != nil {
			return fail(stderr, err)
		}
		return 0
	}

	allowed, err := read(*path)
	if err != nil {
		return fail(stderr, err)
	}
	if over := exceeded(counts, allowed); len(over) > 0 {
		if err := reportExceeded(stdout, over, allowed, current, root); err != nil {
			return fail(stderr, err)
		}
		return 3
	}
	if _, err := fmt.Fprintf(stdout, "antislop: no findings beyond the %d accepted in %s\n", total(allowed), *path); err != nil {
		return fail(stderr, err)
	}
	return 0
}

func fail(stderr io.Writer, err error) int {
	if _, werr := fmt.Fprintln(stderr, "antislop:", err); werr != nil {
		return 1
	}
	return 1
}

// tally counts findings per file and analyzer, with paths relative to root.
func tally(list []findings.Finding, root string) map[key]int {
	out := map[key]int{}
	for _, f := range list {
		out[key{file: rel(root, f.File), analyzer: f.Analyzer}]++
	}
	return out
}

func total(counts map[key]int) int {
	n := 0
	for _, c := range counts {
		n += c
	}
	return n
}

// rel renders a finding's path the way the baseline records it: relative to
// the run directory, with forward slashes.
func rel(root, filename string) string {
	r, err := filepath.Rel(root, filename)
	if err != nil || strings.HasPrefix(r, "..") {
		return filepath.ToSlash(filename)
	}
	return filepath.ToSlash(r)
}

// exceeded returns the pairs whose count is above what the baseline allows,
// sorted for stable output.
func exceeded(counts, allowed map[key]int) []entry {
	var out []entry
	for k, c := range counts {
		if c > allowed[k] {
			out = append(out, entry{count: c, file: k.file, analyzer: k.analyzer})
		}
	}
	slices.SortFunc(out, func(a, b entry) int {
		return cmp.Or(strings.Compare(a.file, b.file), strings.Compare(a.analyzer, b.analyzer))
	})
	return out
}

// reportExceeded prints each pair that grew, followed by the diagnostics it
// covers, so the reader sees the findings rather than only the arithmetic.
func reportExceeded(w io.Writer, over []entry, allowed map[key]int, list []findings.Finding, root string) error {
	for _, e := range over {
		if _, err := fmt.Fprintf(w, "%s: %s: %d finding(s), baseline allows %d\n",
			e.file, e.analyzer, e.count, allowed[key{file: e.file, analyzer: e.analyzer}]); err != nil {
			return err
		}
		var hits []findings.Finding
		for _, f := range list {
			if rel(root, f.File) == e.file && f.Analyzer == e.analyzer {
				hits = append(hits, f)
			}
		}
		slices.SortFunc(hits, func(a, b findings.Finding) int {
			return cmp.Or(cmp.Compare(a.Line, b.Line), strings.Compare(a.Message, b.Message))
		})
		for _, f := range hits {
			if _, err := fmt.Fprintf(w, "\t%s:%d: %s\n", e.file, f.Line, f.Message); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintf(w,
		"\nFindings beyond the baseline. Fix them, or re-record with -update and justify the new entries in review.\n")
	return err
}

// write renders counts to path, sorted by file then analyzer.
func write(path string, counts map[key]int) error {
	entries := make([]entry, 0, len(counts))
	for k, c := range counts {
		entries = append(entries, entry{count: c, file: k.file, analyzer: k.analyzer})
	}
	slices.SortFunc(entries, func(a, b entry) int {
		return cmp.Or(strings.Compare(a.file, b.file), strings.Compare(a.analyzer, b.analyzer))
	})
	var b strings.Builder
	b.WriteString(header)
	for _, e := range entries {
		fmt.Fprintf(&b, "%d %s %s\n", e.count, e.file, e.analyzer)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644) //nolint:gosec // a checked-in lint baseline
}

// read parses path. A missing file is an empty baseline, so the first run of
// a gate on a clean tree succeeds without one.
func read(path string) (map[key]int, error) {
	out := map[key]int{}
	f, err := os.Open(path) //nolint:gosec // path comes from the invoking developer
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	defer f.Close() //nolint:errcheck
	scanner := bufio.NewScanner(f)
	for line := 1; scanner.Scan(); line++ {
		text := scanner.Text()
		if i := strings.IndexByte(text, '#'); i >= 0 {
			text = text[:i]
		}
		fields := strings.Fields(text)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 3 {
			return nil, fmt.Errorf("%s:%d: want '<count> <file> <analyzer>', got %q", path, line, scanner.Text())
		}
		count, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %q is not a count", path, line, fields[0])
		}
		out[key{file: fields[1], analyzer: fields[2]}] += count
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
