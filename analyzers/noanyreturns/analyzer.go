// Package noanyreturns reports function results typed as the empty interface.
package noanyreturns

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sanketsudake/antislop/internal/langver"
	"github.com/sanketsudake/antislop/internal/passutil"
	"github.com/sanketsudake/antislop/internal/seams"
	"github.com/sanketsudake/antislop/internal/typesx"
)

// Name is the analyzer name and diagnostic prefix.
const Name = "noanyreturns"

// Doc describes the analyzer.
const Doc = `reports function results typed as the empty interface (any / interface{})

A result typed any hands the caller a value with no evidence attached: every
caller has to narrow it back with an assertion or a type switch. Return the
type the caller needs, or a small interface listing the methods it calls.

Reported: literal any / interface{} and same-package aliases or named types
whose underlying type is the empty interface, in every function type: func
declarations, func literals, interface methods, and func types in fields,
variables, and type declarations.

Not reported: type parameters (func f[T any]() T), named types from other
packages (database/sql/driver.Value), and functions whose signature is
dictated by an interface or by the slot they are assigned to when
allow-dictated is set (heap.Interface.Pop, flag.Getter.Get,
yaml.Marshaler.MarshalYAML, sync.Pool.New). The dictating interface must be
declared in the same package or in a direct import (or be on the built-in
well-known list); nested func types inside a dictated signature are reported
once, at the contract.

On a concrete method in a file compiled at Go 1.27 or newer, the advice also
names a method type parameter, which returns the caller's own type instead of
erasing it. An interface method keeps the plain advice: interface methods may
not declare type parameters, and a generic method may not implement one.`

// Config holds the analyzer options.
type Config struct {
	// AllowDictated exempts functions whose signature is imposed by an
	// interface or by the slot they are assigned to.
	AllowDictated bool `json:"allow-dictated"`
}

// Default returns the default configuration.
func Default() Config {
	return Config{AllowDictated: true}
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
		Requires: []*analysis.Analyzer{inspect.Analyzer, seams.Analyzer},
		Run:      func(pass *analysis.Pass) (any, error) { run(pass, *c); return nil, nil },
	}
	a.Flags.BoolVar(&c.AllowDictated, "allow-dictated", c.AllowDictated, "allow signatures dictated by an interface or an externally typed slot")
	return a
}

func run(pass *analysis.Pass, cfg Config) {
	// SAFETY: seams.Analyzer declares ResultType *seams.Set and the driver
	// guarantees ResultOf holds a value of that type for every Requires entry.
	dictated := pass.ResultOf[seams.Analyzer].(*seams.Set)
	generated := passutil.GeneratedFiles{}
	passutil.Inspector(pass).WithStack([]ast.Node{(*ast.FuncType)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push || generated.Skip(stack) {
			return true
		}
		// SAFETY: the node filter admits only *ast.FuncType.
		ft := n.(*ast.FuncType)
		if cfg.AllowDictated && isDictated(pass, dictated, stack) {
			// Prune: nested func types inside a dictated signature belong to
			// the contract that dictated it and are reported there.
			return false
		}
		if ft.Results == nil {
			return true
		}
		generic := langver.GenericMethodAdvice(pass, stack)
		for _, field := range ft.Results.List {
			check(pass, field, generic)
		}
		return true
	})
}

// isDictated reports whether the func type belongs to a declaration or literal
// whose signature the author does not control. A func type in a type position
// (a field, a variable, a type declaration) declares the contract itself and is
// therefore never dictated.
func isDictated(pass *analysis.Pass, set *seams.Set, stack []ast.Node) bool {
	if len(stack) < 2 {
		return false
	}
	switch owner := stack[len(stack)-2].(type) {
	case *ast.FuncDecl:
		fn, ok := pass.TypesInfo.Defs[owner.Name].(*types.Func)
		return ok && set.Func(fn)
	case *ast.FuncLit:
		return set.Lit(owner)
	}
	return false
}

// check reports field when its type is the empty interface. genericMethod
// says the enclosing declaration is a concrete method that could take a type
// parameter instead, which adds that alternative to the advice.
func check(pass *analysis.Pass, field *ast.Field, genericMethod bool) {
	t := pass.TypesInfo.TypeOf(field.Type)
	if t == nil || !typesx.IsEmptyInterfaceOwnedBy(t, pass.Pkg) {
		return
	}
	subject := typesx.FieldSubject("result", typesx.ParamNames(field), types.ExprString(field.Type))
	advice := "return a named type (or a small interface with the methods the caller needs)"
	if genericMethod {
		advice += ", or give the method its own type parameter (Go 1.27) and return that"
	}
	pass.Reportf(field.Pos(), "%s: %s, which gives the caller no evidence about the value; %s", Name, subject, advice)
}
