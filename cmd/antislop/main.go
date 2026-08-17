// Command antislop runs the antislop analyzers as a standalone multichecker.
//
//	go get -tool github.com/sanketsudake/antislop/cmd/antislop
//	go tool antislop ./...
//
// Prefer `go tool` (or an installed binary) over `go run`: `go run` reports
// its own exit 1 for any non-zero child status, which collapses "found
// findings" (3) into "failed to run" (1), and it writes progress lines into
// the output stream. Callers that gate on the exit code need them distinct.
//
// Exit codes: 0 no findings, 3 findings reported, 1 an error (bad packages),
// 2 a bad flag.
//
// Disable an analyzer with -<name>=false and set an option with -<name>.<option>.
// Add -summary to print counts per analyzer and per package instead of one
// line per finding, or -baseline <file> to gate against an accepted set;
// -test=false skips test files in any mode. -exclude takes comma-separated
// path patterns whose findings every analyzer skips.
package main

import (
	"flag"
	"os"

	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/sanketsudake/antislop"
	"github.com/sanketsudake/antislop/internal/baseline"
	"github.com/sanketsudake/antislop/internal/exclude"
	"github.com/sanketsudake/antislop/internal/flagx"
	"github.com/sanketsudake/antislop/internal/summary"
)

func main() {
	args := os.Args[1:]
	if summary.Wanted(args) && baseline.Wanted(args) {
		os.Stderr.WriteString("antislop: -summary and -baseline are separate modes; pick one\n") //nolint:errcheck
		os.Exit(2)
	}
	if summary.Wanted(args) {
		os.Exit(summary.Run(args, antislop.Analyzers(), os.Stdout, os.Stderr))
	}
	if baseline.Wanted(args) {
		os.Exit(baseline.Run(args, antislop.Analyzers(), os.Stdout, os.Stderr))
	}
	// multichecker parses flag.CommandLine, so the driver-wide -exclude has
	// to be registered before it runs.
	flag.CommandLine.Var(flagx.NewList(&exclude.Global), "exclude",
		"comma-separated path patterns whose findings every analyzer skips (\"pkg/gen/...\", \"*_test.go\")")
	multichecker.Main(antislop.Analyzers()...)
}
