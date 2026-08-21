// Package langver reports the Go language version in effect for the code
// under analysis, so a diagnostic only names a fix the analyzed package is
// allowed to compile.
package langver

import (
	"go/ast"
	"go/version"

	"golang.org/x/tools/go/analysis"
)

// Go127 is the language version that introduced generic methods.
const Go127 = "go1.27"

// AtLeast reports whether the file at the base of stack is compiled at Go
// language version want ("go1.27") or newer.
//
// The file's own version wins over the package's: a //go:build go1.27 line
// raises a single file above the module's go directive. When neither states a
// version -- a GOPATH-style analysistest fixture is the usual case -- the
// answer is false, so a diagnostic never proposes a fix that the code under
// analysis may not be able to compile.
func AtLeast(pass *analysis.Pass, stack []ast.Node, want string) bool {
	if len(stack) == 0 {
		return false
	}
	file, isFile := stack[0].(*ast.File)
	if !isFile {
		return false
	}
	v := pass.TypesInfo.FileVersions[file]
	if v == "" {
		v = pass.Pkg.GoVersion()
	}
	return version.IsValid(v) && version.Compare(v, want) >= 0
}

// Method returns the method declaration the func type at the top of stack
// belongs to, or nil when the func type is not a concrete method: a plain
// func declaration, a func literal, or a func type in a field, a variable, an
// interface method, or a type declaration.
//
// The distinction carries the generic-method advice. A concrete method may
// declare its own type parameters from Go 1.27; an interface method may not,
// nor may a generic method implement one, so an interface method keeps the
// non-generic wording.
func Method(stack []ast.Node) *ast.FuncDecl {
	if len(stack) < 2 {
		return nil
	}
	decl, isDecl := stack[len(stack)-2].(*ast.FuncDecl)
	if !isDecl || decl.Recv == nil {
		return nil
	}
	return decl
}

// GenericMethodAdvice reports whether a diagnostic about the func type at the
// top of stack may propose a method type parameter: the func type is a
// concrete method declaration, the file is at Go 1.27 or newer, and the
// method does not already declare type parameters.
func GenericMethodAdvice(pass *analysis.Pass, stack []ast.Node) bool {
	decl := Method(stack)
	if decl == nil || decl.Type.TypeParams != nil {
		return false
	}
	return AtLeast(pass, stack, Go127)
}
