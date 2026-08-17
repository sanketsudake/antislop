package narrow

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/sanketsudake/antislop/internal/callspec"
	"github.com/sanketsudake/antislop/internal/typesx"
)

// DefaultSources lists standard-library APIs that hand out untyped values by
// design. Narrowing their immediate result is the boundary of that value, so
// nonarrowany and safetycomment do not report it unless the list is emptied.
var DefaultSources = []string{
	"(context.Context).Value",
	"(*sync.Map).Load",
	"(*sync.Map).LoadAndDelete",
	"(*sync.Map).LoadOrStore",
	"(*sync.Map).Swap",
	"(*sync/atomic.Value).Load",
	"(*sync/atomic.Value).Swap",
	"(*sync.Pool).Get",
	"container/heap.Pop",
	"container/heap.Remove",
	"(*container/list.Element).Value",
	"(*container/ring.Ring).Value",
	"(io/fs.FileInfo).Sys",
}

// Sources is a parsed list of untyped-value sources (see DefaultSources).
type Sources []callspec.Spec

// ParseSources decodes source entries written as "pkg/path.Func",
// "(*pkg/path.Type).Method", or "(pkg/path.Type).Field".
func ParseSources(entries []string) (Sources, error) {
	specs, err := callspec.ParseAll(entries)
	if err != nil {
		return nil, err
	}
	return Sources(specs), nil
}

// SourceMatcher answers, for one pass, whether an operand is the immediate
// result of a listed source. It caches the definitions of local variables per
// function body.
type SourceMatcher struct {
	pass    *analysis.Pass
	sources Sources
	defs    map[*ast.BlockStmt]map[*types.Var]ast.Expr
}

// Matcher builds a per-pass matcher. A nil or empty Sources never matches.
func (s Sources) Matcher(pass *analysis.Pass) *SourceMatcher {
	return &SourceMatcher{pass: pass, sources: s, defs: map[*ast.BlockStmt]map[*types.Var]ast.Expr{}}
}

// Produces reports whether operand -- the operand of a narrowing at the top of
// stack -- is a receipt of an untyped value from a contract that is not this
// package's to change: a call to a listed source, a read of a listed field, a
// field typed any of a struct declared in another package, an element of a
// container of any declared or held by another package
// (jwt.MapClaims["exp"], unstructured.Object["kind"]), or a local variable
// defined from one of these and not reassigned. Foreign fields and elements
// are exempt regardless of the sources list: the declaring package chose any.
func (m *SourceMatcher) Produces(operand ast.Expr, stack []ast.Node) bool {
	if m == nil {
		return false
	}
	return m.produces(operand, stack, 0)
}

const maxDefinitionDepth = 8

func (m *SourceMatcher) produces(operand ast.Expr, stack []ast.Node, depth int) bool {
	if depth > maxDefinitionDepth {
		return false
	}
	switch e := ast.Unparen(operand).(type) {
	case *ast.CallExpr:
		fn, owner, ok := callspec.Callee(m.pass, e)
		return ok && m.matches(fn, owner)
	case *ast.SelectorExpr:
		field, owner, ok := callspec.Field(m.pass, e)
		if !ok {
			return false
		}
		return m.matches(field, owner) || foreignField(m.pass, field)
	case *ast.IndexExpr:
		return foreignElement(m.pass, e, stack, depth, m)
	case *ast.Ident:
		v, isVar := m.pass.TypesInfo.Uses[e].(*types.Var)
		if !isVar || v.Parent() == nil || v.Parent() == m.pass.Pkg.Scope() {
			return false
		}
		body := enclosingBody(stack)
		if body == nil {
			return false
		}
		init, defined := m.definitions(body)[v]
		return defined && init != nil && m.produces(init, stack, depth+1)
	}
	return false
}

func (m *SourceMatcher) matches(obj types.Object, owner string) bool {
	for _, s := range m.sources {
		if s.Matches(obj, owner) {
			return true
		}
	}
	return false
}

// definitions maps each local variable defined in body to the expression it
// was defined from, or nil when it is assigned again anywhere in the body.
func (m *SourceMatcher) definitions(body *ast.BlockStmt) map[*types.Var]ast.Expr {
	if defs, ok := m.defs[body]; ok {
		return defs
	}
	defs := map[*types.Var]ast.Expr{}
	info := m.pass.TypesInfo
	record := func(id *ast.Ident, init ast.Expr) {
		v, ok := info.Defs[id].(*types.Var)
		if !ok {
			return
		}
		defs[v] = init
	}
	taint := func(expr ast.Expr) {
		if id, ok := ast.Unparen(expr).(*ast.Ident); ok {
			if v, ok := info.Uses[id].(*types.Var); ok {
				defs[v] = nil
			}
		}
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			if n.Tok == token.DEFINE {
				for i, lhs := range n.Lhs {
					id, ok := lhs.(*ast.Ident)
					if !ok {
						continue
					}
					if info.Defs[id] == nil { // redeclared in a mixed :=; it is a reassignment
						taint(lhs)
						continue
					}
					record(id, rhsFor(n.Lhs, n.Rhs, i))
				}
			} else {
				for _, lhs := range n.Lhs {
					taint(lhs)
				}
			}
		case *ast.ValueSpec:
			for i, id := range n.Names {
				record(id, rhsFor(exprsOf(n.Names), n.Values, i))
			}
		case *ast.IncDecStmt:
			taint(n.X)
		case *ast.UnaryExpr:
			if n.Op == token.AND {
				taint(n.X)
			}
		case *ast.RangeStmt:
			if n.Tok == token.ASSIGN {
				taint(n.Key)
				taint(n.Value)
			}
		}
		return true
	})
	m.defs[body] = defs
	return defs
}

// rhsFor picks the right-hand side that defines lhs[i]: the positional value
// for a paired assignment, the single tuple-valued call for a, b := f(), or
// the value of a comma-ok map read or assertion for v, ok := m[k] (the ok
// has no defining expression).
func rhsFor(lhs, rhs []ast.Expr, i int) ast.Expr {
	switch {
	case len(rhs) == len(lhs):
		return rhs[i]
	case len(rhs) == 1 && len(lhs) == 2:
		switch ast.Unparen(rhs[0]).(type) {
		case *ast.CallExpr:
			return rhs[0]
		case *ast.IndexExpr, *ast.TypeAssertExpr:
			if i == 0 {
				return rhs[0]
			}
		}
	case len(rhs) == 1:
		if _, isCall := ast.Unparen(rhs[0]).(*ast.CallExpr); isCall {
			return rhs[0]
		}
	}
	return nil
}

func exprsOf(ids []*ast.Ident) []ast.Expr {
	out := make([]ast.Expr, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

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

// foreignField reports whether field is typed any and declared in another package.
func foreignField(pass *analysis.Pass, field *types.Var) bool {
	return field.Pkg() != nil && field.Pkg() != pass.Pkg && typesx.IsEmptyInterface(field.Type())
}

// foreignElement reports whether index reads an element of any out of a
// container that another package declared or holds: a named container type
// from another package (jwt.MapClaims), a field of a foreign struct typed as
// a container of any (unstructured.Unstructured.Object), or a local variable
// defined from either.
func foreignElement(pass *analysis.Pass, index *ast.IndexExpr, stack []ast.Node, depth int, m *SourceMatcher) bool {
	elem := pass.TypesInfo.TypeOf(index)
	if tuple, isTuple := elem.(*types.Tuple); isTuple && tuple.Len() > 0 {
		elem = tuple.At(0).Type() // comma-ok read: v, ok := m[k]
	}
	if !typesx.IsEmptyInterface(elem) {
		return false
	}
	x := ast.Unparen(index.X)
	if foreignNamedContainer(pass, pass.TypesInfo.TypeOf(x)) {
		return true
	}
	switch xe := x.(type) {
	case *ast.SelectorExpr:
		field, _, ok := callspec.Field(pass, xe)
		return ok && field.Pkg() != nil && field.Pkg() != pass.Pkg
	case *ast.Ident:
		v, isVar := pass.TypesInfo.Uses[xe].(*types.Var)
		if !isVar || v.Parent() == nil || v.Parent() == pass.Pkg.Scope() {
			return false
		}
		body := enclosingBody(stack)
		if body == nil {
			return false
		}
		init, defined := m.definitions(body)[v]
		if !defined || init == nil {
			return false
		}
		if sel, ok := ast.Unparen(init).(*ast.SelectorExpr); ok {
			field, _, ok := callspec.Field(pass, sel)
			return ok && field.Pkg() != nil && field.Pkg() != pass.Pkg
		}
		return depth < maxDefinitionDepth && foreignNamedContainer(pass, pass.TypesInfo.TypeOf(init))
	}
	return false
}
