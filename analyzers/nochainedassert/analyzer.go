// Package nochainedassert reports assertions chained through the empty
// interface.
package nochainedassert

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sanketsudake/antislop/internal/narrow"
	"github.com/sanketsudake/antislop/internal/passutil"
	"github.com/sanketsudake/antislop/internal/typesx"
)

// Name is the analyzer name and diagnostic prefix.
const Name = "nochainedassert"

// Doc describes the analyzer.
const Doc = `reports assertions chained through the empty interface (x.(any).(T), any(x).(T))

Widening a value to any and immediately narrowing it back invents evidence the
compiler never had: the conversion throws the declared type away and the
assertion guesses it again. Assert directly from the value's declared type, or
decode the value once at its I/O boundary into a named type.

Reported: a type assertion -- single value, comma-ok, or type switch -- whose
operand is another type assertion, or a conversion to the empty interface
(any(x), interface{}(x), a same-package alias) of a value that is not a type
parameter; and the document-walking shape x.(map[string]any)["k"].(T) or
x.([]any)[i].(T), where an index into a container that was itself asserted
yields another untyped value. The outermost assertion of a chain is reported
once.

Not reported: a single narrowing step (x.(T), switch x.(type), m["k"].(T) on
a plain map of any -- noanycontainers reports the map); the generics
idiom any(v).(T) and switch any(v).(type) where v is a type parameter, which
is the only way to switch on a type parameter; and conversions to a non-empty
interface before an assertion (io.Reader(f).(io.Closer)), which keep the
evidence the value already carried.`

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

func run(pass *analysis.Pass, _ Config) {
	generated := passutil.GeneratedFiles{}
	// An assertion already covered by the diagnostic on the assertion outside
	// it: a chain is one mistake, and is reported at its outermost step.
	covered := map[ast.Node]bool{}
	passutil.Inspector(pass).WithStack([]ast.Node{(*ast.TypeAssertExpr)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push || generated.Skip(stack) {
			return true
		}
		_, operand, ok := narrow.Site(stack)
		if !ok {
			return true
		}
		inner, operandIsAssertion := operand.(*ast.TypeAssertExpr)
		if operandIsAssertion {
			covered[inner] = true
		}
		walked, isWalk := walksDocument(pass, operand)
		if isWalk {
			covered[walked] = true
		}
		if covered[n] {
			return true
		}
		switch {
		case operandIsAssertion || hopsThroughAny(pass, operand):
			pass.Reportf(n.Pos(), "%s: assertion chained through any fabricates evidence; assert directly from the value's declared type, or decode it once at its I/O boundary", Name)
		case isWalk:
			pass.Reportf(n.Pos(), "%s: assertion chained through an any-valued index walks an untyped document; decode it once at its I/O boundary into a struct", Name)
		}
		return true
	})
}

// walksDocument recognises the JSON-walking shape x.(map[string]any)["k"].(T)
// and x.([]any)[i].(T): the operand is an index into a container that was
// itself produced by an assertion, so the element type carries no evidence
// either. It returns the inner assertion so the chain is reported once.
func walksDocument(pass *analysis.Pass, operand ast.Expr) (inner *ast.TypeAssertExpr, ok bool) {
	index, isIndex := ast.Unparen(operand).(*ast.IndexExpr)
	if !isIndex {
		return nil, false
	}
	inner, isAssertion := ast.Unparen(index.X).(*ast.TypeAssertExpr)
	if !isAssertion || !typesx.IsEmptyInterface(pass.TypesInfo.TypeOf(operand)) {
		return nil, false
	}
	return inner, true
}

// hopsThroughAny reports whether operand is a conversion to the empty
// interface of a value that had a type of its own.
func hopsThroughAny(pass *analysis.Pass, operand ast.Expr) bool {
	inner, isConversion := narrow.IsEmptyIfaceConversion(pass, operand)
	return isConversion && !narrow.OperandIsTypeParam(pass, inner)
}
