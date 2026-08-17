// Package nowidenassert reports values widened to the empty interface and
// asserted back in the same function.
package nowidenassert

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sanketsudake/antislop/internal/narrow"
	"github.com/sanketsudake/antislop/internal/passutil"
	"github.com/sanketsudake/antislop/internal/typesx"
)

// Name is the analyzer name and diagnostic prefix.
const Name = "nowidenassert"

// Doc describes the analyzer.
const Doc = `reports values widened to the empty interface and later asserted back in the same function

The widening and the assertion are both in front of the same reader: the
function stored a value it knew the type of, then asked at run time what that
type was. The assertion cannot fail, and it cannot fail for a reason the
compiler could have checked. Keep the concrete type, or thread the small
interface whose methods are used.

Reported: an assertion -- single value, comma-ok, or type switch -- on a local
variable that the same function declared as an empty interface written here
(var v any = x, v := any(x), var v = any(x)) with a concrete initializer, that
is never written again, where the asserted type is the initializer's type or an
interface that type implements. A type switch is reported when one of its cases
names the initializer's type.

Not reported: package-level variables and variables of an enclosing function,
which other code can write; a variable reassigned, ranged over, or whose
address is taken between the declaration and the assertion; a declaration with
no initializer or with an initializer that is already an interface value; an
operand that is not an identifier; a variable whose type is not the empty
interface; type parameters; and an assertion to a type the value could not have
had, which asks a real question instead of repeating a known answer.`

// Config holds the analyzer options. This analyzer has none.
type Config struct{}

// Default returns the default configuration.
func Default() Config { return Config{} }

// Analyzer is the analyzer with default configuration.
var Analyzer = New(Default())

// New builds an analyzer for cfg.
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

// widened records where a local variable took a concrete value into an empty
// interface, and which type that was.
type widened struct {
	decl  *ast.Ident
	known types.Type
}

// scope is a variable seen from one function body: the same variable reads
// differently in the function that declared it and in a closure that captured
// it, so the answer is cached per pair.
type scope struct {
	body *ast.BlockStmt
	v    *types.Var
}

func run(pass *analysis.Pass, _ Config) {
	generated := passutil.GeneratedFiles{}
	seen := map[scope]*widened{}
	passutil.Inspector(pass).WithStack([]ast.Node{(*ast.TypeAssertExpr)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push || generated.Skip(stack) {
			return true
		}
		assert, isAssert := n.(*ast.TypeAssertExpr)
		if !isAssert {
			return true
		}
		kind, operand, ok := narrow.Site(stack)
		if !ok {
			return true
		}
		name, isIdent := operand.(*ast.Ident)
		if !isIdent {
			return true
		}
		v, isVar := pass.TypesInfo.Uses[name].(*types.Var)
		if !isVar {
			return true
		}
		body := enclosingBody(stack)
		if body == nil {
			return true
		}
		key := scope{body: body, v: v}
		w, computed := seen[key]
		if !computed {
			w = widening(pass, body, v)
			seen[key] = w
		}
		if w == nil || w.decl.Pos() >= assert.Pos() {
			return true
		}
		if !assertsBack(pass, kind, stack, assert, w.known) {
			return true
		}
		pass.Reportf(assert.Pos(), "%s: %q was widened to any at line %d and is asserted back here; keep the concrete type instead of widening then asserting",
			Name, name.Name, pass.Fset.Position(w.decl.Pos()).Line)
		return true
	})
}

// enclosingBody returns the body of the innermost function around the node at
// the top of stack. A variable of an outer function was not widened here.
func enclosingBody(stack []ast.Node) *ast.BlockStmt {
	for i := len(stack) - 1; i >= 0; i-- {
		switch fn := stack[i].(type) {
		case *ast.FuncDecl:
			return fn.Body
		case *ast.FuncLit:
			return fn.Body
		}
	}
	return nil
}

// widening reports how v became an empty interface inside body, or nil when v
// was declared elsewhere, was declared without a concrete value, or is written
// again anywhere in body -- in which case the declaration no longer says what
// the variable holds.
func widening(pass *analysis.Pass, body *ast.BlockStmt, v *types.Var) *widened {
	if !typesx.IsEmptyInterfaceOwnedBy(v.Type(), pass.Pkg) {
		return nil
	}
	var found *widened
	written := false
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ValueSpec:
			if len(node.Values) == len(node.Names) {
				for i, name := range node.Names {
					if pass.TypesInfo.Defs[name] == v {
						found = declaration(pass, name, node.Values[i])
					}
				}
			}
		case *ast.AssignStmt:
			paired := len(node.Lhs) == len(node.Rhs)
			for i, lhs := range node.Lhs {
				name, isIdent := lhs.(*ast.Ident)
				switch {
				case !isIdent:
				case pass.TypesInfo.Uses[name] == v:
					written = true
				case pass.TypesInfo.Defs[name] == v && paired:
					found = declaration(pass, name, node.Rhs[i])
				}
			}
		case *ast.IncDecStmt:
			written = written || isVar(pass, node.X, v)
		case *ast.UnaryExpr:
			written = written || (node.Op == token.AND && isVar(pass, node.X, v))
		case *ast.RangeStmt:
			written = written || isVar(pass, node.Key, v) || isVar(pass, node.Value, v)
		}
		return true
	})
	if written {
		return nil
	}
	return found
}

// declaration describes the value a declaration stored, seeing through the
// conversion in v := any(x).
func declaration(pass *analysis.Pass, name *ast.Ident, init ast.Expr) *widened {
	value := init
	if inner, isConversion := narrow.IsEmptyIfaceConversion(pass, init); isConversion {
		value = inner
	}
	if !narrow.IsConcrete(pass, value) {
		return nil
	}
	return &widened{decl: name, known: pass.TypesInfo.TypeOf(value)}
}

// isVar reports whether expr is an identifier already bound to v.
func isVar(pass *analysis.Pass, expr ast.Expr, v *types.Var) bool {
	name, isIdent := expr.(*ast.Ident)
	return isIdent && pass.TypesInfo.Uses[name] == v
}

// assertsBack reports whether the narrowing asks for the type the value is
// already known to have.
func assertsBack(pass *analysis.Pass, kind narrow.Kind, stack []ast.Node, assert *ast.TypeAssertExpr, known types.Type) bool {
	if kind == narrow.TypeSwitch {
		return switchNamesType(pass, stack, known)
	}
	asserted := pass.TypesInfo.TypeOf(assert.Type)
	if asserted == nil {
		return false
	}
	if types.Identical(known, asserted) {
		return true
	}
	iface, isIface := asserted.Underlying().(*types.Interface)
	return isIface && types.Implements(known, iface)
}

// switchNamesType reports whether one of the cases of the enclosing type switch
// names the type the value is known to have. A switch that lists only other
// types narrows nothing back.
func switchNamesType(pass *analysis.Pass, stack []ast.Node, known types.Type) bool {
	for i := len(stack) - 1; i >= 0; i-- {
		stmt, isSwitch := stack[i].(*ast.TypeSwitchStmt)
		if !isSwitch {
			continue
		}
		for _, clause := range stmt.Body.List {
			cases, isCase := clause.(*ast.CaseClause)
			if !isCase {
				continue
			}
			for _, expr := range cases.List {
				if types.Identical(known, pass.TypesInfo.TypeOf(expr)) {
					return true
				}
			}
		}
		return false
	}
	return false
}
