// Package nostructuralnames reports identifiers that contain a forbidden
// structural term.
package nostructuralnames

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sanketsudake/antislop/internal/flagx"
	"github.com/sanketsudake/antislop/internal/passutil"
)

// Name is the analyzer name and diagnostic prefix.
const Name = "nostructuralnames"

// Doc describes the analyzer.
const Doc = `reports identifiers that contain a forbidden structural term (default: shape)

A name like UserShape or reshape describes how a value is arranged rather than
what it is for. It fits any type with the same fields, so it tells the next
reader nothing that can be checked against the domain, and it survives every
refactoring that changes the meaning. Name the symbol after the role it plays
-- Order, Invoice, Retries -- and let its type describe the structure.

Reported: every identifier this package declares whose name contains one of
the terms, matched case-insensitively as a substring: the package clause, type
names, function and method names, var and const names, struct fields,
interface methods, parameters, results, receivers, type parameters, labels,
short variable declarations, and range variables. When several terms match,
the diagnostic names the first one in the list.

Not reported: uses of a symbol declared somewhere else (an imported geo.Shape
is that package's naming decision, and so is a method of ours that implements
an external interface -- only the declaration is ours to rename), the names
files give to their imports, and the blank identifier.

Setting terms replaces the default list rather than adding to it.`

// Config holds the analyzer options.
type Config struct {
	// Terms lists the structural terms to reject, matched case-insensitively
	// as a substring. Setting it replaces the default list.
	Terms []string `json:"terms"`
}

// Default returns the default configuration.
func Default() Config {
	return Config{Terms: []string{"shape"}}
}

// Analyzer is the analyzer with default configuration and flag bindings.
var Analyzer = New(Default())

// New builds an analyzer for cfg. Flags are bound to a copy of cfg so the
// standalone driver can override options.
func New(cfg Config) *analysis.Analyzer {
	c := &cfg
	a := &analysis.Analyzer{
		Name:     Name,
		Doc:      Doc,
		URL:      "https://github.com/sanketsudake/antislop#" + Name,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      func(pass *analysis.Pass) (any, error) { run(pass, *c); return nil, nil },
	}
	a.Flags.Var(flagx.NewList(&c.Terms), "terms", "comma-separated structural terms to reject in declared names (replaces the default list)")
	return a
}

func run(pass *analysis.Pass, cfg Config) {
	generated := passutil.GeneratedFiles{}
	passutil.Inspector(pass).WithStack([]ast.Node{(*ast.Ident)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push || generated.Skip(stack) {
			return true
		}
		name, isIdent := n.(*ast.Ident)
		if !isIdent || !declares(pass, name) {
			return true
		}
		term, forbidden := match(cfg.Terms, name.Name)
		if !forbidden {
			return true
		}
		pass.Reportf(name.Pos(), "%s: name %q describes structure (%q) rather than the domain role; rename it for what it represents",
			Name, name.Name, term)
		return true
	})
}

// declares reports whether name introduces a symbol this package is free to
// rename. The package clause is recorded with no object and counts; an import
// name renames another package rather than declaring anything here; the blank
// identifier names nothing.
func declares(pass *analysis.Pass, name *ast.Ident) bool {
	if name.Name == "_" {
		return false
	}
	obj, defined := pass.TypesInfo.Defs[name]
	if !defined {
		return false
	}
	_, isImport := obj.(*types.PkgName)
	return !isImport
}

// match returns the first term that appears in name, compared without regard
// to case so Shape, shape, and reshape all count.
func match(terms []string, name string) (term string, forbidden bool) {
	lower := strings.ToLower(name)
	for _, t := range terms {
		if t != "" && strings.Contains(lower, strings.ToLower(t)) {
			return t, true
		}
	}
	return "", false
}
