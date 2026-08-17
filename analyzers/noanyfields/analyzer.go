// Package noanyfields reports struct fields typed as the empty interface.
package noanyfields

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sanketsudake/antislop/internal/passutil"
	"github.com/sanketsudake/antislop/internal/typesx"
)

// Name is the analyzer name and diagnostic prefix.
const Name = "noanyfields"

// Doc describes the analyzer.
const Doc = `reports struct fields typed as the empty interface (any / interface{})

A field typed any turns the struct into a bag: nothing states what the value
is, so every reader narrows it back with an assertion or a type switch. Give
the field the type it really holds, or split the struct into one type per
variant.

Reported: literal any / interface{} and same-package aliases or named types
whose underlying type is the empty interface, in every struct type: type
declarations and anonymous structs in variables, parameters, results, and
composite literal types. An embedded field of such a type is reported too.

Not reported: type parameter fields (struct{ v T }), named types from other
packages (database/sql/driver.Value), func-typed fields (noanyparams reports
the parameter), and container-typed fields such as map[string]any
(noanycontainers reports the container).

This analyzer has no options.`

// Config holds the analyzer options. This analyzer has none.
type Config struct{}

// Default returns the default configuration.
func Default() Config {
	return Config{}
}

// Analyzer is the analyzer with default configuration.
var Analyzer = New(Default())

// New builds an analyzer for cfg. The configuration is empty; New exists so
// the registry can build every analyzer the same way.
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

func run(pass *analysis.Pass, _ Config) {
	generated := passutil.GeneratedFiles{}
	passutil.Inspector(pass).WithStack([]ast.Node{(*ast.StructType)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push || generated.Skip(stack) {
			return true
		}
		// SAFETY: the node filter admits only *ast.StructType.
		st := n.(*ast.StructType)
		if st.Fields == nil {
			return true
		}
		for _, field := range st.Fields.List {
			check(pass, field)
		}
		return true
	})
}

func check(pass *analysis.Pass, field *ast.Field) {
	t := pass.TypesInfo.TypeOf(field.Type)
	if t == nil || !typesx.IsEmptyInterfaceOwnedBy(t, pass.Pkg) {
		return
	}
	subject := typesx.FieldSubject("field", fieldNames(field), types.ExprString(field.Type))
	pass.Reportf(field.Type.Pos(), "%s: %s, which carries no evidence about the value; give the field the type it actually holds, or split the struct per variant", Name, subject)
}

// fieldNames returns the names a struct field declares. An embedded field
// declares no name of its own: it is named after its type.
func fieldNames(field *ast.Field) []string {
	if len(field.Names) > 0 {
		return typesx.ParamNames(field)
	}
	if name := embeddedName(field.Type); name != "" {
		return []string{name}
	}
	return nil
}

func embeddedName(expr ast.Expr) string {
	switch e := ast.Unparen(expr).(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.StarExpr:
		return embeddedName(e.X)
	case *ast.IndexExpr:
		return embeddedName(e.X)
	case *ast.IndexListExpr:
		return embeddedName(e.X)
	}
	return ""
}
