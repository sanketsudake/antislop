// Package nonarrowany reports checked narrowing of an empty-interface value.
package nonarrowany

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sanketsudake/antislop/internal/flagx"
	"github.com/sanketsudake/antislop/internal/narrow"
	"github.com/sanketsudake/antislop/internal/passutil"
	"github.com/sanketsudake/antislop/internal/seams"
	"github.com/sanketsudake/antislop/internal/typesx"
)

// Name is the analyzer name and diagnostic prefix.
const Name = "nonarrowany"

// Doc describes the analyzer.
const Doc = `reports checked narrowing of an empty-interface value (comma-ok assertions and type switches)

A comma-ok assertion or a type switch on any is the Go analogue of
typeof x === "string": it tests a representation instead of establishing a
contract. The value arrived without evidence, and asking what it happens to be
does not add any. Decode it once at its I/O boundary into a named type and
branch on that type, or accept a small interface with the methods you call.

Reported: v, ok := x.(T) and switch x.(type) (with or without a binding) when
the operand is structurally the empty interface, whichever package the name
came from: any, interface{}, a same-package alias, and named types such as
database/sql/driver.Value.

Not reported: single-value assertions x.(T), which safetycomment covers;
narrowing of a non-empty interface, which tests a contract the value already
carries (err.(net.Error), switch n := node.(type) on ast.Node); the generics
idiom any(v).(T) and switch any(v).(type) where v is a type parameter; and,
when allow-in-parse-funcs is set, narrowing inside a boundary function that
takes an empty-interface parameter and returns error or bool as its last
result (Scan(src any) error, parse(v any) (User, error)); and narrowing the
immediate result of a source in the sources list -- standard-library APIs that
hand out untyped values by design (context.Context.Value, sync.Map.Load,
atomic.Value.Load, sync.Pool.Get, heap.Pop, list.Element.Value), reached
directly or through a local variable defined from the call and not reassigned.
Narrowing there is the boundary of that value. Setting sources replaces the
default list; an empty list reports every narrowing.

Two more receipts of a foreign contract are never reported: a parameter typed
any of a function whose signature is dictated (a callback slot typed by another
package such as sync.Map.Range or an informer handler, an interface method),
because the author cannot retype it; and a field typed any of a struct declared
in another package, or an element of a container of any that another package
declared or holds (cache.DeletedFinalStateUnknown.Obj, jwt.MapClaims["exp"],
unstructured.Object["kind"]), reached directly or through a local variable,
because that package chose any. With skip-declared-any
set, narrowing a value read from a declaration this package owns -- a
parameter typed any, a field typed any, an element of a container of any --
is not reported either: noanyparams, noanyfields, or noanycontainers already
reports the declaration, and one finding per decision reads better than one
per use.`

// Config holds the analyzer options.
type Config struct {
	// AllowInParseFuncs exempts narrowing inside a decode boundary: a function
	// that takes an any parameter and reports failure through a final error or
	// bool result.
	AllowInParseFuncs bool `json:"allow-in-parse-funcs"`
	// Sources lists standard-library APIs whose immediate result may be
	// narrowed without a report, as "pkg/path.Func", "(*pkg/path.Type).Method"
	// or "(pkg/path.Type).Field". Setting it replaces the default list.
	Sources []string `json:"sources"`
	// SkipDeclaredAny leaves narrowing of a value read from a declaration
	// this package owns -- a parameter typed any, a field typed any, an
	// element of a container of any -- to the analyzer that reports the
	// declaration: one finding per decision instead of one per use.
	SkipDeclaredAny bool `json:"skip-declared-any"`
}

// Default returns the default configuration.
func Default() Config {
	return Config{AllowInParseFuncs: false, Sources: slices.Clone(narrow.DefaultSources)}
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
		Run: func(pass *analysis.Pass) (any, error) {
			sources, err := narrow.ParseSources(c.Sources)
			if err != nil {
				return nil, fmt.Errorf("%s: invalid sources entry: %w", Name, err)
			}
			run(pass, *c, sources.Matcher(pass), dictatedSet(pass))
			return nil, nil
		},
	}
	a.Flags.BoolVar(&c.AllowInParseFuncs, "allow-in-parse-funcs", c.AllowInParseFuncs,
		"allow narrowing inside a boundary function that takes an any parameter and returns error or bool")
	a.Flags.BoolVar(&c.SkipDeclaredAny, "skip-declared-any", c.SkipDeclaredAny, "leave narrowing of a parameter, field, or container element typed any declared in this package to the analyzer that reports the declaration")
	a.Flags.Var(flagx.NewList(&c.Sources), "sources", "comma-separated standard-library sources of untyped values whose immediate result may be narrowed (replaces the default list)")
	return a
}

func run(pass *analysis.Pass, cfg Config, sources *narrow.SourceMatcher, dictated *seams.Set) {
	generated := passutil.GeneratedFiles{}
	passutil.Inspector(pass).WithStack([]ast.Node{(*ast.TypeAssertExpr)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push || generated.Skip(stack) {
			return true
		}
		kind, operand, ok := narrow.Site(stack)
		if !ok || kind == narrow.Unchecked {
			return true
		}
		if !typesx.IsEmptyInterface(pass.TypesInfo.TypeOf(operand)) {
			return true
		}
		if inner, isHop := narrow.IsEmptyIfaceConversion(pass, operand); isHop && narrow.OperandIsTypeParam(pass, inner) {
			return true
		}
		if cfg.AllowInParseFuncs && inParseFunc(pass, stack) {
			return true
		}
		if sources.Produces(operand, stack) || narrow.IsDictatedParam(pass, dictated, operand, stack) {
			return true
		}
		if cfg.SkipDeclaredAny && narrow.IsDeclaredAny(pass, operand) {
			return true
		}
		form := "comma-ok assertion"
		if kind == narrow.TypeSwitch {
			form = "type switch"
		}
		pass.Reportf(n.Pos(), "%s: %s on any narrows a representation without establishing its contract; decode the value at its I/O boundary into a named type and branch on that", Name, form)
		return true
	})
}

// inParseFunc reports whether the innermost function enclosing the narrowing
// site is a decode boundary: it takes an empty-interface parameter and its
// last result is error or bool.
func inParseFunc(pass *analysis.Pass, stack []ast.Node) bool {
	for i := len(stack) - 1; i >= 0; i-- {
		var ft *ast.FuncType
		switch fn := stack[i].(type) {
		case *ast.FuncDecl:
			ft = fn.Type
		case *ast.FuncLit:
			ft = fn.Type
		default:
			continue
		}
		return takesAny(pass, ft) && reportsFailure(pass, ft)
	}
	return false
}

func takesAny(pass *analysis.Pass, ft *ast.FuncType) bool {
	if ft.Params == nil {
		return false
	}
	for _, field := range ft.Params.List {
		expr := field.Type
		if el, isVariadic := expr.(*ast.Ellipsis); isVariadic {
			expr = el.Elt
		}
		if typesx.IsEmptyInterface(pass.TypesInfo.TypeOf(expr)) {
			return true
		}
	}
	return false
}

// reportsFailure reports whether the last result of ft is error or bool, the
// two ways a decode boundary says "this input did not fit".
func reportsFailure(pass *analysis.Pass, ft *ast.FuncType) bool {
	if ft.Results == nil || len(ft.Results.List) == 0 {
		return false
	}
	last := ft.Results.List[len(ft.Results.List)-1]
	t := pass.TypesInfo.TypeOf(last.Type)
	if t == nil {
		return false
	}
	if types.Identical(t, types.Universe.Lookup("error").Type()) {
		return true
	}
	basic, isBasic := t.Underlying().(*types.Basic)
	return isBasic && basic.Kind() == types.Bool
}

func dictatedSet(pass *analysis.Pass) *seams.Set {
	// SAFETY: seams.Analyzer declares ResultType *seams.Set and the driver
	// guarantees ResultOf holds a value of that type for every Requires entry.
	return pass.ResultOf[seams.Analyzer].(*seams.Set)
}
