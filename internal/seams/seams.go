// Package seams computes, once per package, the set of functions whose
// signature is dictated by an external contract rather than chosen by the
// author. Analyzers that reject `any` in parameters or results consult this
// set so that a signature the author cannot change is not reported.
//
// A function is dictated when:
//
//  1. it is a method whose name and signature match a method of an interface
//     declared in the same package or in a direct import (heap.Interface,
//     sql.Scanner, driver.Valuer, context.Context.Value, ...);
//  2. it is a method matching a well-known standard-library contract whose
//     defining package is often not imported by the implementer
//     (Scan(any) error, As(any) bool, Value(any) any, MarshalYAML() (any, error), ...);
//  3. it is a func literal, or a package-level func referenced by name, that
//     is used as a value in a slot whose type is declared elsewhere: a
//     composite-literal field or element, a call argument (unless the
//     parameter type is inferred from the argument itself), the right-hand
//     side of an assignment to a field or typed variable, or a return value.
//     The contract lives at the slot's declaration, which is reported there
//     when it belongs to the package under analysis.
package seams

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sanketsudake/antislop/internal/passutil"
)

// Set answers whether a function or func literal has a dictated signature.
type Set struct {
	funcs map[*types.Func]bool
	lits  map[*ast.FuncLit]bool
}

// Func reports whether fn (a method or package-level func) has a dictated signature.
func (s *Set) Func(fn *types.Func) bool { return s != nil && s.funcs[fn] }

// Lit reports whether the func literal has a dictated signature.
func (s *Set) Lit(lit *ast.FuncLit) bool { return s != nil && s.lits[lit] }

// Analyzer computes the *Set for a package.
var Analyzer = &analysis.Analyzer{
	Name:       "antislopseams",
	Doc:        "computes the set of functions whose signature is dictated by an external contract (internal helper for antislop)",
	Requires:   []*analysis.Analyzer{inspect.Analyzer},
	ResultType: reflect.TypeOf((*Set)(nil)),
	Run:        run,
}

func run(pass *analysis.Pass) (any, error) {
	s := &Set{funcs: map[*types.Func]bool{}, lits: map[*ast.FuncLit]bool{}}
	ifaces := interfaceMethods(pass.Pkg)
	insp := passutil.Inspector(pass)

	insp.WithStack([]ast.Node{(*ast.FuncDecl)(nil), (*ast.FuncLit)(nil), (*ast.Ident)(nil)},
		func(n ast.Node, push bool, stack []ast.Node) bool {
			if !push {
				return true
			}
			switch n := n.(type) {
			case *ast.FuncDecl:
				fn, ok := pass.TypesInfo.Defs[n.Name].(*types.Func)
				if ok && n.Recv != nil && (ifaces.matches(fn) || wellKnown(fn)) {
					s.funcs[fn] = true
				}
			case *ast.FuncLit:
				if slotDictated(pass, stack) {
					s.lits[n] = true
				}
			case *ast.Ident:
				fn, ok := pass.TypesInfo.Uses[n].(*types.Func)
				if !ok || fn.Pkg() != pass.Pkg || fn.Signature().Recv() != nil {
					return true
				}
				if slotDictated(pass, stack) {
					s.funcs[fn] = true
				}
			}
			return true
		})
	return s, nil
}

// slotDictated inspects the parent chain of the func value at stack[len-1]
// and reports whether the slot it occupies has a declared type of its own.
func slotDictated(pass *analysis.Pass, stack []ast.Node) bool {
	// SAFETY: the node filter admits only *ast.FuncLit and *ast.Ident, both ast.Expr.
	value := stack[len(stack)-1].(ast.Expr)
	i := len(stack) - 2
	for i >= 0 {
		paren, ok := stack[i].(*ast.ParenExpr)
		if !ok {
			break
		}
		value = paren
		i--
	}
	if i < 0 {
		return false
	}
	parent := stack[i]
	switch p := parent.(type) {
	case *ast.KeyValueExpr:
		if p.Value != value || i == 0 {
			return false
		}
		_, ok := stack[i-1].(*ast.CompositeLit)
		return ok
	case *ast.CompositeLit:
		return true
	case *ast.CallExpr:
		if p.Fun == value {
			return false
		}
		return argSlotDictated(pass, p, value)
	case *ast.AssignStmt:
		if p.Tok != token.ASSIGN {
			return false
		}
		for _, l := range p.Lhs {
			if id, ok := l.(*ast.Ident); ok && id.Name == "_" {
				return false
			}
		}
		return true
	case *ast.ValueSpec:
		return p.Type != nil
	case *ast.ReturnStmt:
		return true
	}
	return false
}

// argSlotDictated reports whether the parameter receiving arg has a type
// that does not depend on a type parameter inferred from the argument.
func argSlotDictated(pass *analysis.Pass, call *ast.CallExpr, arg ast.Expr) bool {
	var callee types.Object
	switch f := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		callee = pass.TypesInfo.Uses[f]
	case *ast.SelectorExpr:
		callee = pass.TypesInfo.Uses[f.Sel]
	case *ast.IndexExpr, *ast.IndexListExpr:
		return true // explicit instantiation: the slot type is spelled out
	default:
		// Calling a func value: its type is declared at the value's own slot.
		return true
	}
	if callee == nil {
		return false
	}
	if _, isType := callee.(*types.TypeName); isType {
		return true // conversion to a declared func type
	}
	sig, ok := callee.Type().(*types.Signature)
	if !ok {
		return true
	}
	if sig.TypeParams().Len() == 0 {
		return true
	}
	idx := -1
	for j, a := range call.Args {
		if a == arg {
			idx = j
		}
	}
	if idx < 0 {
		return false
	}
	params := sig.Params()
	if idx >= params.Len() {
		if !sig.Variadic() {
			return false
		}
		idx = params.Len() - 1
	}
	return !mentionsTypeParam(params.At(idx).Type())
}

func mentionsTypeParam(t types.Type) bool {
	found := false
	var visit func(types.Type)
	seen := map[types.Type]bool{}
	visit = func(t types.Type) {
		if found || t == nil || seen[t] {
			return
		}
		seen[t] = true
		switch tt := t.(type) {
		case *types.TypeParam:
			found = true
		case *types.Alias:
			visit(tt.Rhs())
		case *types.Pointer:
			visit(tt.Elem())
		case *types.Slice:
			visit(tt.Elem())
		case *types.Array:
			visit(tt.Elem())
		case *types.Chan:
			visit(tt.Elem())
		case *types.Map:
			visit(tt.Key())
			visit(tt.Elem())
		case *types.Signature:
			visit(tt.Params())
			visit(tt.Results())
		case *types.Tuple:
			for i := range tt.Len() {
				visit(tt.At(i).Type())
			}
		case *types.Struct:
			for i := range tt.NumFields() {
				visit(tt.Field(i).Type())
			}
		case *types.Named:
			for i := range tt.TypeArgs().Len() {
				visit(tt.TypeArgs().At(i))
			}
		}
	}
	visit(t)
	return found
}

// interfaceMethods indexes interface method signatures by name from the
// package itself and its direct imports.
type methodIndex map[string][]*types.Signature

func interfaceMethods(pkg *types.Package) methodIndex {
	idx := methodIndex{}
	add := func(scope *types.Scope) {
		for _, name := range scope.Names() {
			tn, ok := scope.Lookup(name).(*types.TypeName)
			if !ok {
				continue
			}
			iface, ok := tn.Type().Underlying().(*types.Interface)
			if !ok {
				continue
			}
			for i := range iface.NumMethods() {
				m := iface.Method(i)
				idx[m.Name()] = append(idx[m.Name()], m.Signature())
			}
		}
	}
	add(pkg.Scope())
	for _, imp := range pkg.Imports() {
		add(imp.Scope())
	}
	return idx
}

func (idx methodIndex) matches(fn *types.Func) bool {
	sig := fn.Signature()
	for _, cand := range idx[fn.Name()] {
		if types.Identical(sig, cand) {
			return true
		}
	}
	return false
}

// wellKnown matches methods against standard-library contracts whose defining
// package the implementer frequently does not import.
func wellKnown(fn *types.Func) bool {
	sig := fn.Signature()
	for _, cand := range wellKnownSignatures[fn.Name()] {
		if types.Identical(sig, cand) {
			return true
		}
	}
	return false
}

var wellKnownSignatures = func() map[string][]*types.Signature {
	anyT := types.Universe.Lookup("any").Type()
	errT := types.Universe.Lookup("error").Type()
	boolT := types.Typ[types.Bool]
	v := func(name string, t types.Type) *types.Var { return types.NewVar(0, nil, name, t) }
	sig := func(params, results []*types.Var) *types.Signature {
		return types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), types.NewTuple(results...), false)
	}
	return map[string][]*types.Signature{
		"Scan":        {sig([]*types.Var{v("src", anyT)}, []*types.Var{v("", errT)})},     // database/sql.Scanner
		"As":          {sig([]*types.Var{v("target", anyT)}, []*types.Var{v("", boolT)})}, // errors.As target contract
		"Value":       {sig([]*types.Var{v("key", anyT)}, []*types.Var{v("", anyT)})},     // context.Context
		"Push":        {sig([]*types.Var{v("x", anyT)}, nil)},                             // container/heap.Interface
		"Pop":         {sig(nil, []*types.Var{v("", anyT)})},                              // container/heap.Interface
		"Get":         {sig(nil, []*types.Var{v("", anyT)})},                              // flag.Getter
		"MarshalYAML": {sig(nil, []*types.Var{v("", anyT), v("", errT)})},                 // yaml.Marshaler
	}
}()
