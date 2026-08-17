// Package narrow classifies type-assertion sites for the antislop analyzers
// that reason about narrowing: which shape tests the type (comma-ok, type
// switch) and which one asserts it (single value), what the narrowed operand
// is, and whether that operand only became an interface through a conversion.
package narrow

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/sanketsudake/antislop/internal/callspec"
	"github.com/sanketsudake/antislop/internal/seams"
	"github.com/sanketsudake/antislop/internal/typesx"
)

// Kind says how a type assertion tests its operand.
type Kind int

const (
	// Unchecked is a single-value assertion x.(T): it panics on mismatch.
	Unchecked Kind = iota
	// CommaOK is the two-value form v, ok := x.(T).
	CommaOK
	// TypeSwitch is the x.(type) operand of a type switch statement.
	TypeSwitch
)

// Site classifies the *ast.TypeAssertExpr at the top of stack (the node an
// inspector.WithStack callback was handed) and returns the narrowed operand
// with its parentheses removed. ok is false when the top of stack is not a
// type assertion.
func Site(stack []ast.Node) (kind Kind, operand ast.Expr, ok bool) {
	if len(stack) == 0 {
		return Unchecked, nil, false
	}
	assert, isAssert := stack[len(stack)-1].(*ast.TypeAssertExpr)
	if !isAssert {
		return Unchecked, nil, false
	}
	operand = ast.Unparen(assert.X)
	switch {
	case assert.Type == nil: // x.(type) only exists inside a type switch
		return TypeSwitch, operand, true
	case isCommaOK(stack, assert):
		return CommaOK, operand, true
	default:
		return Unchecked, operand, true
	}
}

// isCommaOK reports whether assert, the node at the top of stack, is the sole
// right-hand side of a two-value assignment or declaration.
func isCommaOK(stack []ast.Node, assert ast.Expr) bool {
	child := assert
	for i := len(stack) - 2; i >= 0; i-- {
		switch parent := stack[i].(type) {
		case *ast.ParenExpr:
			child = parent
		case *ast.AssignStmt:
			return len(parent.Lhs) == 2 && len(parent.Rhs) == 1 && parent.Rhs[0] == child
		case *ast.ValueSpec:
			return len(parent.Names) == 2 && len(parent.Values) == 1 && parent.Values[0] == child
		default:
			return false
		}
	}
	return false
}

// IsEmptyIfaceConversion reports whether expr is a conversion to the empty
// interface -- any(x), interface{}(x), or a named type or alias that resolves
// to it -- and returns the converted operand.
func IsEmptyIfaceConversion(pass *analysis.Pass, expr ast.Expr) (inner ast.Expr, ok bool) {
	call, isCall := ast.Unparen(expr).(*ast.CallExpr)
	if !isCall || len(call.Args) != 1 || call.Ellipsis.IsValid() {
		return nil, false
	}
	tv, known := pass.TypesInfo.Types[ast.Unparen(call.Fun)]
	if !known || !tv.IsType() || !typesx.IsEmptyInterface(tv.Type) {
		return nil, false
	}
	return ast.Unparen(call.Args[0]), true
}

// OperandIsTypeParam reports whether expr has the type of a type parameter.
// Converting one to any is the only way to switch on it, so that hop is an
// idiom rather than a lost type.
func OperandIsTypeParam(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	return t != nil && typesx.IsTypeParam(t)
}

// IsConcrete reports whether expr already carries evidence of what it holds:
// its static type is not an interface, not a type parameter, and not the
// untyped nil. An untyped constant counts as its default type (int, string),
// which is what widening it to any would throw away.
func IsConcrete(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	// A type parameter is checked first: its underlying type is its
	// constraint interface, so types.IsInterface would say yes.
	if t == nil || typesx.IsTypeParam(t) {
		return false
	}
	if basic, isBasic := types.Unalias(t).(*types.Basic); isBasic && basic.Kind() == types.UntypedNil {
		return false
	}
	return !types.IsInterface(t)
}

// IsDeclaredAny reports whether operand reads an untyped value out of a
// declaration this package owns and another analyzer already reports: a
// parameter typed any (noanyparams), a field typed any of a struct declared
// here (noanyfields), or an element of a container of any declared here
// (noanycontainers). Analyzers may choose to leave the narrowing of such a
// value to that report, so one decision yields one finding.
func IsDeclaredAny(pass *analysis.Pass, operand ast.Expr) bool {
	switch e := ast.Unparen(operand).(type) {
	case *ast.Ident:
		v, ok := pass.TypesInfo.Uses[e].(*types.Var)
		return ok && v.Kind() == types.ParamVar && typesx.IsEmptyInterfaceOwnedBy(v.Type(), pass.Pkg)
	case *ast.SelectorExpr:
		field, _, ok := callspec.Field(pass, e)
		return ok && field.Pkg() == pass.Pkg && typesx.IsEmptyInterfaceOwnedBy(field.Type(), pass.Pkg)
	case *ast.IndexExpr:
		return ownedContainer(pass, pass.TypesInfo.TypeOf(e.X))
	}
	return false
}

// ownedContainer reports whether t is a map, slice or array of an owned empty
// interface, either written out (map[string]any) or named in this package.
func ownedContainer(pass *analysis.Pass, t types.Type) bool {
	if t == nil {
		return false
	}
	if ptr, isPtr := types.Unalias(t).(*types.Pointer); isPtr {
		t = ptr.Elem()
	}
	if named, isNamed := types.Unalias(t).(*types.Named); isNamed {
		if named.Obj().Pkg() != nil && named.Obj().Pkg() != pass.Pkg {
			return false
		}
	}
	switch u := t.Underlying().(type) {
	case *types.Map:
		return typesx.IsEmptyInterfaceOwnedBy(u.Elem(), pass.Pkg)
	case *types.Slice:
		return typesx.IsEmptyInterfaceOwnedBy(u.Elem(), pass.Pkg)
	case *types.Array:
		return typesx.IsEmptyInterfaceOwnedBy(u.Elem(), pass.Pkg)
	}
	return false
}

// IsDictatedParam reports whether operand is a parameter, typed structurally
// as the empty interface, of an enclosing function whose signature is dictated
// by an external contract (see internal/seams): a callback slot typed by
// another package, an interface method. The author cannot retype such a
// parameter, so narrowing it is the boundary of that contract.
func IsDictatedParam(pass *analysis.Pass, dictated *seams.Set, operand ast.Expr, stack []ast.Node) bool {
	id, ok := ast.Unparen(operand).(*ast.Ident)
	if !ok {
		return false
	}
	v, ok := pass.TypesInfo.Uses[id].(*types.Var)
	if !ok || v.Kind() != types.ParamVar || !typesx.IsEmptyInterface(v.Type()) {
		return false
	}
	for i := len(stack) - 1; i >= 0; i-- {
		var ft *ast.FuncType
		var owned bool
		switch fn := stack[i].(type) {
		case *ast.FuncLit:
			ft, owned = fn.Type, dictated.Lit(fn)
		case *ast.FuncDecl:
			obj, isFunc := pass.TypesInfo.Defs[fn.Name].(*types.Func)
			ft, owned = fn.Type, isFunc && dictated.Func(obj)
		default:
			continue
		}
		if declaresParam(pass, ft, v) {
			return owned
		}
	}
	return false
}

func declaresParam(pass *analysis.Pass, ft *ast.FuncType, v *types.Var) bool {
	if ft == nil || ft.Params == nil {
		return false
	}
	for _, field := range ft.Params.List {
		for _, name := range field.Names {
			if pass.TypesInfo.Defs[name] == v {
				return true
			}
		}
	}
	return false
}

func foreignNamedContainer(pass *analysis.Pass, t types.Type) bool {
	if t == nil {
		return false
	}
	if ptr, isPtr := types.Unalias(t).(*types.Pointer); isPtr {
		t = ptr.Elem()
	}
	named, isNamed := types.Unalias(t).(*types.Named)
	if !isNamed || named.Obj().Pkg() == nil || named.Obj().Pkg() == pass.Pkg {
		return false
	}
	switch u := named.Underlying().(type) {
	case *types.Map:
		return typesx.IsEmptyInterface(u.Elem())
	case *types.Slice:
		return typesx.IsEmptyInterface(u.Elem())
	case *types.Array:
		return typesx.IsEmptyInterface(u.Elem())
	}
	return false
}
