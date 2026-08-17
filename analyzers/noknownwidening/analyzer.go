// Package noknownwidening reports known concrete values stored into
// empty-interface locations.
package noknownwidening

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sanketsudake/antislop/internal/narrow"
	"github.com/sanketsudake/antislop/internal/passutil"
	"github.com/sanketsudake/antislop/internal/typesx"
)

// Name is the analyzer name and diagnostic prefix.
const Name = "noknownwidening"

// Doc describes the analyzer.
const Doc = `reports known concrete values stored into empty-interface locations

The compiler knew what the value was, and the code threw that away on purpose:
storing an int in an any turns a fact into a guess that every later reader has
to re-establish with an assertion. Keep the concrete type, or name the small
interface whose methods are actually used.

Reported, when the location is the empty interface written here (any,
interface{}, or a same-package alias or named type) and the value is concrete
(its static type is not an interface, not a type parameter, and not nil):
a var declared with that type and an initializer (var v any = 42), per
name/value pair; an assignment to a variable or to a struct field declared in
this package (v = 42, s.Payload = u), paired positionally; and an explicit
conversion (any(x), interface{}(x), Owned(x)).

Not reported: a var with no initializer (var v any); a target that is not the
empty interface, including another package's contract (driver.Value); an
initializer that is already an interface value (var v any = err); assignments
into containers (m["k"] = 42), which noanycontainers owns; returns and
composite-literal fields, which noanyreturns and noanyfields own; a concrete
argument passed to a parameter typed any (fmt.Println(42)), which noanyparams
owns at the signature; any(v) on a type parameter, the only way to inspect T;
the operand of an assertion (any(x).(T)), which nochainedassert owns; and,
while allow-variadic-args is set, a conversion forwarded to a variadic ...any
parameter or written as an element of a []any literal.`

// Config holds the analyzer options.
type Config struct {
	// AllowVariadicArgs exempts a conversion that is an argument to a variadic
	// ...any parameter or an element of a []any literal: the fmt / slog idiom.
	AllowVariadicArgs bool `json:"allow-variadic-args"`
}

// Default returns the default configuration.
func Default() Config {
	return Config{AllowVariadicArgs: true}
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
	a.Flags.BoolVar(&c.AllowVariadicArgs, "allow-variadic-args", c.AllowVariadicArgs,
		"allow a conversion forwarded to a variadic ...any parameter or written as an element of a []any literal")
	return a
}

func run(pass *analysis.Pass, cfg Config) {
	generated := passutil.GeneratedFiles{}
	filter := []ast.Node{(*ast.ValueSpec)(nil), (*ast.AssignStmt)(nil), (*ast.CallExpr)(nil)}
	passutil.Inspector(pass).WithStack(filter, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push || generated.Skip(stack) {
			return true
		}
		switch node := n.(type) {
		case *ast.ValueSpec:
			checkValueSpec(pass, node)
		case *ast.AssignStmt:
			checkAssign(pass, node)
		case *ast.CallExpr:
			checkConversion(pass, cfg, node, stack)
		}
		return true
	})
}

// checkValueSpec reports the initializers of a var declared with an
// empty-interface type. Names and values pair up positionally; a single value
// for several names is a tuple-returning call, which has no pairing to judge.
func checkValueSpec(pass *analysis.Pass, spec *ast.ValueSpec) {
	if spec.Type == nil || len(spec.Values) != len(spec.Names) {
		return
	}
	if !typesx.IsEmptyInterfaceOwnedBy(pass.TypesInfo.TypeOf(spec.Type), pass.Pkg) {
		return
	}
	for _, value := range spec.Values {
		report(pass, value)
	}
}

// checkAssign reports the right-hand sides that land in an empty-interface
// location. A single right-hand side for several locations is a
// tuple-returning call.
func checkAssign(pass *analysis.Pass, assign *ast.AssignStmt) {
	if assign.Tok != token.ASSIGN && assign.Tok != token.DEFINE {
		return
	}
	if len(assign.Lhs) != len(assign.Rhs) {
		return
	}
	for i, target := range assign.Lhs {
		if isEmptyIfaceLocation(pass, target) {
			report(pass, assign.Rhs[i])
		}
	}
}

// isEmptyIfaceLocation reports whether target names a variable or a struct
// field declared in this package whose type is the empty interface. Index
// expressions and pointer indirections are deliberately not locations here:
// a container of any is noanycontainers' report.
func isEmptyIfaceLocation(pass *analysis.Pass, target ast.Expr) bool {
	var name *ast.Ident
	switch expr := ast.Unparen(target).(type) {
	case *ast.Ident:
		name = expr
	case *ast.SelectorExpr:
		name = expr.Sel
	default:
		return false
	}
	if name.Name == "_" {
		return false
	}
	v, isVar := pass.TypesInfo.ObjectOf(name).(*types.Var)
	if !isVar || v.Pkg() != pass.Pkg {
		return false
	}
	return typesx.IsEmptyInterfaceOwnedBy(v.Type(), pass.Pkg)
}

// checkConversion reports an explicit conversion of a concrete value to the
// empty interface, unless the conversion only forwards the value to an API
// that asks for any, or feeds an assertion that narrows it straight back.
func checkConversion(pass *analysis.Pass, cfg Config, call *ast.CallExpr, stack []ast.Node) {
	inner, isConversion := narrow.IsEmptyIfaceConversion(pass, call)
	if !isConversion || !typesx.IsEmptyInterfaceOwnedBy(pass.TypesInfo.TypeOf(call), pass.Pkg) {
		return
	}
	parent, child := parentOf(stack, call)
	if isAssertedOperand(parent, child) {
		return
	}
	if cfg.AllowVariadicArgs && isForwarded(pass, parent, child) {
		return
	}
	report(pass, inner)
}

// parentOf returns the nearest ancestor of expr that is not a parenthesis,
// together with the child of that ancestor holding expr (expr itself, or the
// outermost parenthesis around it).
func parentOf(stack []ast.Node, expr ast.Expr) (parent ast.Node, child ast.Expr) {
	child = expr
	for i := len(stack) - 2; i >= 0; i-- {
		paren, isParen := stack[i].(*ast.ParenExpr)
		if !isParen {
			return stack[i], child
		}
		child = paren
	}
	return nil, child
}

// isAssertedOperand reports whether child is the operand of a type assertion
// or of a type switch: any(x).(T) is nochainedassert's report, not ours.
func isAssertedOperand(parent ast.Node, child ast.Expr) bool {
	assert, isAssert := parent.(*ast.TypeAssertExpr)
	return isAssert && assert.X == child
}

// isForwarded reports whether child only hands the value to an API that asks
// for any: a variadic ...any parameter, or the []any literal such a call takes.
func isForwarded(pass *analysis.Pass, parent ast.Node, child ast.Expr) bool {
	switch node := parent.(type) {
	case *ast.CallExpr:
		return isVariadicAnyArg(pass, node, child)
	case *ast.CompositeLit:
		return isAnySequenceElement(pass, node, child)
	}
	return false
}

// isVariadicAnyArg reports whether arg lands in a variadic ...any parameter.
func isVariadicAnyArg(pass *analysis.Pass, call *ast.CallExpr, arg ast.Expr) bool {
	if call.Ellipsis.IsValid() {
		return false
	}
	fun := pass.TypesInfo.TypeOf(call.Fun)
	if fun == nil {
		return false
	}
	sig, isSig := fun.Underlying().(*types.Signature)
	if !isSig || !sig.Variadic() {
		return false
	}
	last := sig.Params().Len() - 1
	slice, isSlice := sig.Params().At(last).Type().Underlying().(*types.Slice)
	if !isSlice || !typesx.IsEmptyInterface(slice.Elem()) {
		return false
	}
	index := slices.Index(call.Args, arg)
	return index >= last
}

// isAnySequenceElement reports whether elem is written directly in a []any (or
// [N]any) literal, the argument such APIs are built from.
func isAnySequenceElement(pass *analysis.Pass, lit *ast.CompositeLit, elem ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(lit)
	if t == nil {
		return false
	}
	var element types.Type
	switch underlying := t.Underlying().(type) {
	case *types.Slice:
		element = underlying.Elem()
	case *types.Array:
		element = underlying.Elem()
	default:
		return false
	}
	return typesx.IsEmptyInterface(element) && slices.Contains(lit.Elts, elem)
}

// report states the type the code chose to forget, unless the value never had
// one to begin with.
func report(pass *analysis.Pass, value ast.Expr) {
	if !narrow.IsConcrete(pass, value) {
		return
	}
	written := types.TypeString(pass.TypesInfo.TypeOf(value), types.RelativeTo(pass.Pkg))
	pass.Reportf(value.Pos(), "%s: value of type %s is stored as any, discarding what is known about it; keep the concrete type (or a small interface with the methods that are used)", Name, written)
}
