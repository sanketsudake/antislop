// Package callspec parses and matches textual references to functions,
// methods, and fields of named types, as used by list-typed analyzer options:
//
//	pkg/path.Func                 a package-level function
//	(*pkg/path.Type).Method       a method (pointer or value receiver)
//	(pkg/path.Type).Member        a method or a field of the named type
//
// The receiver's pointerness is written for readability only; matching
// ignores it, so "(*sync.Map).Load" and "(sync.Map).Load" name the same method.
package callspec

import (
	"errors"
	"go/ast"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Spec names a package-level function or a member (method or field) of a
// named type.
type Spec struct {
	Pkg  string // import path of the declaring package
	Recv string // named type the member belongs to; empty for a package-level function
	Name string // function, method, or field name
}

// ErrSyntax reports a malformed entry.
var ErrSyntax = errors.New("callspec: malformed entry")

// Parse decodes one entry.
func Parse(entry string) (Spec, error) {
	if !strings.HasPrefix(entry, "(") {
		pkg, name, ok := splitQualified(entry)
		if !ok {
			return Spec{}, ErrSyntax
		}
		return Spec{Pkg: pkg, Name: name}, nil
	}
	end := strings.Index(entry, ")")
	if end < 0 || !strings.HasPrefix(entry[end+1:], ".") {
		return Spec{}, ErrSyntax
	}
	pkg, recv, ok := splitQualified(strings.TrimPrefix(entry[1:end], "*"))
	if !ok {
		return Spec{}, ErrSyntax
	}
	name := entry[end+2:]
	if !isIdent(name) {
		return Spec{}, ErrSyntax
	}
	return Spec{Pkg: pkg, Recv: recv, Name: name}, nil
}

// ParseAll decodes every entry, stopping at the first malformed one.
func ParseAll(entries []string) ([]Spec, error) {
	out := make([]Spec, 0, len(entries))
	for _, entry := range entries {
		s, err := Parse(entry)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// SplitTarget separates a trailing "#N" argument index from an entry:
// "encoding/json.Unmarshal#1" yields ("encoding/json.Unmarshal", 1).
func SplitTarget(entry string) (spec string, target int, err error) {
	hash := strings.LastIndex(entry, "#")
	if hash < 0 {
		return "", 0, ErrSyntax
	}
	target, convErr := strconv.Atoi(entry[hash+1:])
	if convErr != nil || target < 0 {
		return "", 0, ErrSyntax
	}
	return entry[:hash], target, nil
}

// splitQualified cuts "pkg/path.Name" at its last dot. Import paths may hold
// dots (gopkg.in/yaml.v3) but the trailing identifier may not.
func splitQualified(text string) (pkg, name string, ok bool) {
	dot := strings.LastIndex(text, ".")
	if dot <= 0 {
		return "", "", false
	}
	pkg, name = text[:dot], text[dot+1:]
	return pkg, name, isIdent(name)
}

func isIdent(name string) bool {
	return name != "" && !strings.ContainsAny(name, "./*()#")
}

// Matches reports whether obj -- a package-level function, a method, or a
// field -- is what the spec names. owner is the named type the member belongs
// to (see Owner), empty for package-level functions.
func (s Spec) Matches(obj types.Object, owner string) bool {
	return obj != nil && obj.Pkg() != nil && s.Pkg == obj.Pkg().Path() && s.Name == obj.Name() && s.Recv == owner
}

// Callee resolves the function or method a call invokes and the name of the
// receiver's named type (empty for a package-level function). Method
// expressions ((*T).M) and calls through func values are not resolved.
func Callee(pass *analysis.Pass, call *ast.CallExpr) (fn *types.Func, owner string, ok bool) {
	switch fun := ast.Unparen(call.Fun).(type) {
	case *ast.SelectorExpr:
		if selection := pass.TypesInfo.Selections[fun]; selection != nil {
			if selection.Kind() != types.MethodVal {
				return nil, "", false
			}
			method, isFunc := selection.Obj().(*types.Func)
			if !isFunc {
				return nil, "", false
			}
			return method, Owner(method), true
		}
		return packageFunc(pass, fun.Sel)
	case *ast.Ident:
		return packageFunc(pass, fun)
	}
	return nil, "", false
}

// Field resolves a selector that reads a struct field and returns the field
// and the name of the named struct type that declares it.
func Field(pass *analysis.Pass, sel *ast.SelectorExpr) (field *types.Var, owner string, ok bool) {
	selection := pass.TypesInfo.Selections[sel]
	if selection == nil || selection.Kind() != types.FieldVal {
		return nil, "", false
	}
	v, isVar := selection.Obj().(*types.Var)
	if !isVar {
		return nil, "", false
	}
	// The receiver type of the selection is the type the selector was
	// applied to; strip pointers and aliases to find the named struct.
	return v, namedName(selection.Recv()), true
}

func packageFunc(pass *analysis.Pass, name *ast.Ident) (fn *types.Func, owner string, ok bool) {
	f, isFunc := pass.TypesInfo.Uses[name].(*types.Func)
	if !isFunc || f.Signature().Recv() != nil {
		return nil, "", false
	}
	return f, "", true
}

// Owner is the name of the named type a method is declared on, with any
// pointer stripped: (*json.Decoder).Decode reports "Decoder".
func Owner(fn *types.Func) string {
	recv := fn.Signature().Recv()
	if recv == nil {
		return ""
	}
	return namedName(recv.Type())
}

func namedName(t types.Type) string {
	t = types.Unalias(t)
	if ptr, isPtr := t.(*types.Pointer); isPtr {
		t = types.Unalias(ptr.Elem())
	}
	named, isNamed := t.(*types.Named)
	if !isNamed {
		return ""
	}
	return named.Obj().Name()
}

// Display names a function the way a reader writes it: json.Unmarshal,
// json.Decoder.Decode.
func Display(fn *types.Func, owner string) string {
	name := fn.Pkg().Name() + "."
	if owner != "" {
		name += owner + "."
	}
	return name + fn.Name()
}
