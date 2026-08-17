// Package noanytypes reports type declarations that rename the empty interface.
package noanytypes

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sanketsudake/antislop/internal/passutil"
	"github.com/sanketsudake/antislop/internal/typesx"
)

// Name is the analyzer name and diagnostic prefix.
const Name = "noanytypes"

// Doc describes the analyzer.
const Doc = `reports type declarations that merely rename the empty interface

A type whose right-hand side is the empty interface adds a name and nothing
else: values of that type still carry no evidence, and every use of the name
has to narrow them back. Declare the fields or the methods the values really
have, or delete the declaration.

Reported: declarations and aliases whose right-hand side is the empty
interface written literally (type X = any, type X any, type X interface{}),
an interface that embeds any and nothing else, or another same-package name
that resolves to the empty interface. Every declaration in such a chain is
reported at its own name, and a generic alias counts (type Box[T any] = any).

Not reported: constraint interfaces (comparable, type sets such as
~int | ~float64), interfaces with methods, aliases of a type parameter, and
declarations whose right-hand side is a named type from another package
(database/sql/driver.Value), which owns its own contract.

This analyzer has no options.`

// Config holds the analyzer options. This analyzer has none.
type Config struct{}

// Default returns the default configuration.
func Default() Config {
	return Config{}
}

// Analyzer is the analyzer with default configuration.
var Analyzer = New(Default())

// New builds an analyzer for cfg. The configuration is empty; New exists so
// the registry can build every analyzer the same way.
func New(cfg Config) *analysis.Analyzer {
	c := &cfg
	return &analysis.Analyzer{
		Name:     Name,
		Doc:      Doc,
		URL:      "https://github.com/sanketsudake/antislop#" + Name,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      func(pass *analysis.Pass) (any, error) { run(pass, *c); return nil, nil },
	}
}

func run(pass *analysis.Pass, _ Config) {
	generated := passutil.GeneratedFiles{}
	passutil.Inspector(pass).WithStack([]ast.Node{(*ast.TypeSpec)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push || generated.Skip(stack) {
			return true
		}
		// SAFETY: the node filter admits only *ast.TypeSpec.
		spec := n.(*ast.TypeSpec)
		// The right-hand side decides: a declaration that renames an imported
		// type inherits that package's contract instead of erasing one.
		t := pass.TypesInfo.TypeOf(spec.Type)
		if t == nil || !typesx.IsEmptyInterfaceOwnedBy(t, pass.Pkg) {
			return true
		}
		pass.Reportf(spec.Name.Pos(), "%s: type %q is the empty interface under another name and carries no evidence; declare the fields or methods the values actually have, or delete the alias", Name, spec.Name.Name)
		return true
	})
}
