// Package langver reports the Go language version in effect for the code
// under analysis, so a diagnostic only names a fix the analyzed package is
// allowed to compile.
package langver

import (
	"go/ast"
	"go/version"

	"golang.org/x/tools/go/analysis"
)

// go127 is the language version that introduced generic methods.
const go127 = "go1.27"

// GenericMethodAdvice reports whether a diagnostic about the func type at the
// top of stack may propose a method type parameter: the func type is a
// concrete method declaration, the file is at Go 1.27 or newer, and the
// method does not already declare type parameters.
//
// The concrete-method distinction carries the advice. An interface method may
// not declare type parameters, nor may a generic method implement one, so an
// interface method -- like a plain func declaration, a func literal, or a
// func type in a field, a variable, or a type declaration -- keeps the
// non-generic wording.
func GenericMethodAdvice(pass *analysis.Pass, stack []ast.Node) bool {
	decl := method(stack)
	if decl == nil || decl.Type.TypeParams != nil {
		return false
	}
	// SAFETY: inspector.WithStack roots every stack at the enclosing *ast.File.
	file := stack[0].(*ast.File)
	return atLeast(pass.TypesInfo.FileVersions[file], pass.Pkg.GoVersion(), go127)
}

// method returns the method declaration the func type at the top of stack
// belongs to, or nil when the func type is not a concrete method.
func method(stack []ast.Node) *ast.FuncDecl {
	if len(stack) < 2 {
		return nil
	}
	decl, isDecl := stack[len(stack)-2].(*ast.FuncDecl)
	if !isDecl || decl.Recv == nil {
		return nil
	}
	return decl
}

// atLeast reports whether the code the file/pkg versions describe is compiled
// at Go language version want ("go1.27") or newer. The file's own version
// wins over the package's: a //go:build go1.27 line raises a single file
// above the module's go directive. When neither states a version -- a
// GOPATH-style analysistest fixture is the usual case -- the answer is false,
// so a diagnostic never proposes a fix that the code under analysis may not
// be able to compile.
func atLeast(file, pkg, want string) bool {
	v := file
	if v == "" {
		v = pkg
	}
	return version.IsValid(v) && version.Compare(v, want) >= 0
}
