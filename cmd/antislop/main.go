// Command antislop runs the antislop analyzers as a standalone multichecker.
//
//	go run github.com/sanketsudake/antislop/cmd/antislop@latest ./...
//
// Disable an analyzer with -<name>=false and set an option with -<name>.<option>.
// Add -summary to print counts per analyzer and per package instead of one
// line per finding; -test=false skips test files in either mode.
package main

import (
	"os"

	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/sanketsudake/antislop"
	"github.com/sanketsudake/antislop/internal/summary"
)

func main() {
	if summary.Wanted(os.Args[1:]) {
		os.Exit(summary.Run(os.Args[1:], antislop.Analyzers(), os.Stdout, os.Stderr))
	}
	multichecker.Main(antislop.Analyzers()...)
}
