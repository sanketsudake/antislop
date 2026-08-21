// Package noanyparams reports function parameters typed as the empty interface.
package noanyparams

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
const Name = "noanyparams"

// Doc describes the analyzer.
const Doc = `reports parameters typed as the empty interface (any / interface{})

A parameter typed any pushes decoding onto every callee and carries no
evidence about the value. Decode input once at its I/O boundary into a named
type and accept that type instead.

Reported: literal any / interface{} and same-package aliases or named types
whose underlying type is the empty interface, in every function type: func
declarations, func literals, interface methods, and func types in fields,
variables, and type declarations.

Not reported: type parameters (T any), named types from other packages
(database/sql/driver.Value), variadic ...any when allow-variadic is set, and
methods whose signature is dictated by an interface or a slot declared
elsewhere when allow-dictated is set (heap.Interface.Push, sql.Scanner.Scan,
sync.Pool.New). The dictating interface must be declared in the same package
or in a direct import (or be on the built-in well-known list such as
Scan(any) error); nested func types inside a dictated signature are reported
once, at the contract.

On a concrete method in a file compiled at Go 1.27 or newer, the advice also
names a method type parameter, which keeps the caller's type through the call
instead of erasing it. An interface method keeps the plain advice, since
interface methods may not declare type parameters. A generic method may not
implement an interface either, so the alternative holds only for a method no
interface dictates -- and that is the author's call, not this rule's: a
consumer package may declare an interface this rule cannot see.`

// Config holds the analyzer options.
type Config struct {
	// AllowVariadic exempts variadic parameters (args ...any), the fmt / slog idiom.
	AllowVariadic bool `json:"allow-variadic"`
	// AllowDictated exempts functions whose signature is imposed by an
	// interface or by the slot they are assigned to.
	AllowDictated bool `json:"allow-dictated"`
}

// Default returns the default configuration.
func Default() Config {
	return Config{AllowVariadic: true, AllowDictated: true}
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
	a.Flags.BoolVar(&c.AllowVariadic, "allow-variadic", c.AllowVariadic, "allow variadic ...any parameters")
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
		if ft.Params == nil {
			return true
		}
		generic := langver.GenericMethodAdvice(pass, stack)
		for _, field := range ft.Params.List {
			check(pass, cfg, field, generic)
		}
		return true
	})
}

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
func check(pass *analysis.Pass, cfg Config, field *ast.Field, genericMethod bool) {
	typeExpr := field.Type
	if el, ok := typeExpr.(*ast.Ellipsis); ok {
		if cfg.AllowVariadic {
			return
		}
		typeExpr = el.Elt
	}
	t := pass.TypesInfo.TypeOf(typeExpr)
	if t == nil || !typesx.IsEmptyInterfaceOwnedBy(t, pass.Pkg) {
		return
	}
	subject := typesx.FieldSubject("parameter", typesx.ParamNames(field), types.ExprString(typeExpr))
	advice := "decode input at its I/O boundary into a named type and accept that type"
	if genericMethod {
		advice += ", or give the method its own type parameter (Go 1.27) so the caller's type survives the call"
	}
	pass.Reportf(field.Pos(), "%s: %s, which carries no evidence about the value; %s", Name, subject, advice)
}
