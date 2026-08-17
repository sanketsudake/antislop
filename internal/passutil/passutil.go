// Package passutil holds small helpers shared by every antislop analyzer.
package passutil

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Inspector returns the shared AST inspector produced by inspect.Analyzer.
// Every antislop analyzer lists inspect.Analyzer in Requires.
func Inspector(pass *analysis.Pass) *inspector.Inspector {
	// SAFETY: inspect.Analyzer declares ResultType *inspector.Inspector and the
	// driver guarantees ResultOf holds a value of that type for every Requires entry.
	return pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
}

// FileOf returns the *ast.File of pass that contains pos, or nil.
func FileOf(pass *analysis.Pass, pos token.Pos) *ast.File {
	for _, f := range pass.Files {
		if f.FileStart <= pos && pos < f.FileEnd {
			return f
		}
	}
	return nil
}

// IsTestFile reports whether pos lies in a *_test.go file.
func IsTestFile(pass *analysis.Pass, pos token.Pos) bool {
	return strings.HasSuffix(pass.Fset.File(pos).Name(), "_test.go")
}

// IsGenerated reports whether pos lies in a generated file
// ("// Code generated ... DO NOT EDIT.").
func IsGenerated(pass *analysis.Pass, pos token.Pos) bool {
	f := FileOf(pass, pos)
	return f != nil && ast.IsGenerated(f)
}

// Files returns the non-generated files of pass.
func Files(pass *analysis.Pass) []*ast.File {
	out := make([]*ast.File, 0, len(pass.Files))
	for _, f := range pass.Files {
		if !ast.IsGenerated(f) {
			out = append(out, f)
		}
	}
	return out
}

// GeneratedFiles memoises ast.IsGenerated per file for stack-based walks.
type GeneratedFiles map[*ast.File]bool

// Skip reports whether the walk rooted at stack[0] is inside a generated file.
func (g GeneratedFiles) Skip(stack []ast.Node) bool {
	if len(stack) == 0 {
		return false
	}
	f, ok := stack[0].(*ast.File)
	if !ok {
		return false
	}
	v, seen := g[f]
	if !seen {
		v = ast.IsGenerated(f)
		g[f] = v
	}
	return v
}
