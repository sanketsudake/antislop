// Package typesx holds type predicates shared by antislop analyzers.
package typesx

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"
)

// IsEmptyInterface reports whether t is structurally the empty interface
// (any / interface{}) after unwrapping aliases and named types from any
// package. Type parameters are never empty interfaces: their constraint is
// not the type of the value.
func IsEmptyInterface(t types.Type) bool {
	return isEmpty(t, nil)
}

// IsEmptyInterfaceOwnedBy reports whether t is the empty interface written
// literally (any / interface{}) or through an alias or named type declared
// in pkg. A named type from another package (for example database/sql/driver.Value)
// is that package's contract and is not reported at its use site.
func IsEmptyInterfaceOwnedBy(t types.Type, pkg *types.Package) bool {
	return isEmpty(t, pkg)
}

func isEmpty(t types.Type, owner *types.Package) bool {
	for {
		switch tt := t.(type) {
		case *types.Alias:
			if owner != nil && foreign(tt.Obj(), owner) {
				return false
			}
			t = tt.Rhs()
		case *types.Named:
			if owner != nil && foreign(tt.Obj(), owner) {
				return false
			}
			t = tt.Underlying()
		case *types.Interface:
			return tt.Empty()
		default:
			return false
		}
	}
}

func foreign(obj *types.TypeName, owner *types.Package) bool {
	return obj.Pkg() != nil && obj.Pkg() != owner
}

// IsTypeParam reports whether t (after unwrapping aliases) is a type parameter.
func IsTypeParam(t types.Type) bool {
	_, ok := types.Unalias(t).(*types.TypeParam)
	return ok
}

// IsVariadicField reports whether f is a variadic parameter (`args ...T`).
func IsVariadicField(f *ast.Field) bool {
	_, ok := f.Type.(*ast.Ellipsis)
	return ok
}

// ParamNames returns the names declared by a parameter field, or a single
// empty string when the parameter is unnamed.
func ParamNames(f *ast.Field) []string {
	if len(f.Names) == 0 {
		return []string{""}
	}
	out := make([]string, len(f.Names))
	for i, n := range f.Names {
		out[i] = n.Name
	}
	return out
}

// FieldSubject renders the diagnostic subject for a field of a func type or a
// struct: "result has type any" when the field is unnamed, "result \"v\" has
// type any" for one name, and "results \"a\", \"b\" have type any" for several.
// noun is the singular form; the plural adds an "s".
func FieldSubject(noun string, names []string, written string) string {
	switch {
	case len(names) == 0 || names[0] == "":
		return noun + " has type " + written
	case len(names) == 1:
		return fmt.Sprintf("%s %q has type %s", noun, names[0], written)
	default:
		quoted := make([]string, len(names))
		for i, n := range names {
			quoted[i] = fmt.Sprintf("%q", n)
		}
		return fmt.Sprintf("%ss %s have type %s", noun, strings.Join(quoted, ", "), written)
	}
}
