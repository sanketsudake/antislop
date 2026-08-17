// Package summary implements the standalone binary's -summary mode: it runs
// the analyzers over the requested packages and prints counts per analyzer
// and per package instead of one line per finding. It is the first thing to
// run on an existing codebase, to decide what to enable and where to start.
package summary

import (
	"cmp"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

// Wanted reports whether args request summary mode.
func Wanted(args []string) bool {
	for _, a := range args {
		if a == "-summary" || a == "--summary" {
			return true
		}
		if a == "--" {
			break
		}
	}
	return false
}

// Run executes summary mode and returns the process exit code: 0 with no
// findings, 3 with findings (as the multichecker does), 1 on error.
func Run(args []string, analyzers []*analysis.Analyzer, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("antislop -summary", flag.ContinueOnError)
	fs.SetOutput(stderr)
	_ = fs.Bool("summary", true, "print counts per analyzer and package instead of findings")
	tests := fs.Bool("test", true, "analyze test files too")
	top := fs.Int("top", 15, "number of packages to list (0 for all)")
	dir := fs.String("dir", "", "directory to run in (default: current)")
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

	counts := aggregate(graph)
	if err := counts.print(stdout, *top); err != nil {
		return fail(stderr, err)
	}
	if counts.total == 0 {
		return 0
	}
	return 3
}

type perAnalyzer struct {
	name           string
	total, inTests int
	files          map[string]bool
	packages       map[string]bool
}

type table struct {
	total    int
	inTests  int
	analyzer map[string]*perAnalyzer
	pkg      map[string]int
	seen     map[string]bool // dedup across the test and non-test variants of a package
}

func fail(stderr io.Writer, err error) int {
	if _, werr := fmt.Fprintln(stderr, "antislop:", err); werr != nil {
		return 1
	}
	return 1
}

func aggregate(graph *checker.Graph) *table {
	t := &table{analyzer: map[string]*perAnalyzer{}, pkg: map[string]int{}, seen: map[string]bool{}}
	for act := range graph.All() {
		if !act.IsRoot {
			continue
		}
		fset := act.Package.Fset
		for _, d := range act.Diagnostics {
			pos := fset.Position(d.Pos)
			key := act.Analyzer.Name + "\x00" + pos.String() + "\x00" + d.Message
			if t.seen[key] {
				continue
			}
			t.seen[key] = true
			pa := t.analyzer[act.Analyzer.Name]
			if pa == nil {
				pa = &perAnalyzer{name: act.Analyzer.Name, files: map[string]bool{}, packages: map[string]bool{}}
				t.analyzer[act.Analyzer.Name] = pa
			}
			pa.total++
			pa.files[pos.Filename] = true
			pkgPath := act.Package.PkgPath
			pkgPath = strings.TrimSuffix(pkgPath, "_test")
			pkgPath = strings.TrimSuffix(pkgPath, ".test")
			pa.packages[pkgPath] = true
			t.pkg[pkgPath]++
			t.total++
			if strings.HasSuffix(pos.Filename, "_test.go") {
				pa.inTests++
				t.inTests++
			}
		}
	}
	return t
}

func (t *table) print(w io.Writer, top int) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	rows := make([]*perAnalyzer, 0, len(t.analyzer))
	for _, pa := range t.analyzer {
		rows = append(rows, pa)
	}
	slices.SortFunc(rows, func(a, b *perAnalyzer) int {
		return cmp.Or(cmp.Compare(b.total, a.total), cmp.Compare(a.name, b.name))
	})
	lines := []string{"analyzer\tfindings\tin tests\tfiles\tpackages"}
	for _, pa := range rows {
		lines = append(lines, fmt.Sprintf("%s\t%d\t%d\t%d\t%d", pa.name, pa.total, pa.inTests, len(pa.files), len(pa.packages)))
	}
	lines = append(lines, fmt.Sprintf("total\t%d\t%d\t\t", t.total, t.inTests))
	if err := writeLines(tw, lines); err != nil {
		return err
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if len(t.pkg) == 0 {
		return nil
	}

	type pkgRow struct {
		path  string
		count int
	}
	prow := make([]pkgRow, 0, len(t.pkg))
	for p, n := range t.pkg {
		prow = append(prow, pkgRow{p, n})
	}
	slices.SortFunc(prow, func(a, b pkgRow) int {
		return cmp.Or(cmp.Compare(b.count, a.count), cmp.Compare(a.path, b.path))
	})
	if top > 0 && len(prow) > top {
		prow = prow[:top]
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	lines = []string{"package\tfindings"}
	for _, r := range prow {
		lines = append(lines, fmt.Sprintf("%s\t%d", r.path, r.count))
	}
	if err := writeLines(tw, lines); err != nil {
		return err
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if top > 0 && len(t.pkg) > top {
		_, err := fmt.Fprintf(w, "(%d more packages; -top=0 lists all)\n", len(t.pkg)-top)
		return err
	}
	return nil
}

func writeLines(w io.Writer, lines []string) error {
	_, err := io.WriteString(w, strings.Join(lines, "\n")+"\n")
	return err
}
