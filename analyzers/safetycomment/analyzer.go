// Package safetycomment requires a stated invariant in front of every
// unchecked escape hatch.
package safetycomment

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sanketsudake/antislop/internal/comments"
	"github.com/sanketsudake/antislop/internal/flagx"
	"github.com/sanketsudake/antislop/internal/narrow"
	"github.com/sanketsudake/antislop/internal/passutil"
	"github.com/sanketsudake/antislop/internal/seams"
)

// Name is the analyzer name and diagnostic prefix.
const Name = "safetycomment"

// Doc describes the analyzer.
const Doc = `requires a SAFETY comment before every unchecked escape hatch: single-value type assertions, unsafe conversions, and go:linkname

Each of these asserts an invariant the compiler cannot see, and fails at run
time when the invariant does not hold. The escape hatch is sometimes the right
answer; leaving the reason unwritten never is. State the checked invariant in a
"// SAFETY:" comment on the statement above, or inline just before the
expression, so the next reader can verify it.

Reported: a single-value type assertion x.(T) on any interface operand, empty
or not, because it panics on mismatch; a conversion to or from unsafe.Pointer;
a call to unsafe.Add, unsafe.Slice, unsafe.SliceData, unsafe.String or
unsafe.StringData; and every //go:linkname directive.

Not reported: any of the above with a SAFETY comment, comma-ok assertions and
type switches, which state the check in code, the compile-time queries
unsafe.Sizeof, unsafe.Alignof and unsafe.Offsetof, and an assertion on the
immediate result of a source in the sources list -- standard-library APIs
that hand out untyped values by design (context.Context.Value, sync.Map.Load,
atomic.Value.Load, sync.Pool.Get, heap.Pop, list.Element.Value), reached
directly or through a local variable defined from the call and not
reassigned. Setting sources replaces the default list; an empty list requires
SAFETY everywhere. Two receipts of a foreign contract need no SAFETY comment
either: a parameter typed any of a function whose signature is dictated (a
callback slot typed by another package, an interface method), and a field
typed any of a struct declared elsewhere or an element of a container of any
that another package declared or holds (jwt.MapClaims["exp"],
unstructured.Object["kind"]), reached directly or through a local variable --
that package chose any, and the assertion is the boundary. With skip-declared-any set, an unchecked assertion on a value read
from a declaration this package owns -- a parameter typed any, a field typed
any, an element of a container of any -- is not reported either: noanyparams,
noanyfields, or noanycontainers already reports the declaration.

A justification counts when it ends before the escape hatch begins and sits on
the line above the owning statement or declaration, or inline on the same line
(/* SAFETY: ... */ x.(T)). A doc comment on the enclosing function does not
count: it is too far away to say which invariant one line relies on. For
//go:linkname the comment belongs in the directive's own comment group.`

// Config holds the analyzer options.
type Config struct {
	// Sources lists standard-library APIs whose immediate result may be
	// asserted without a SAFETY comment, as "pkg/path.Func",
	// "(*pkg/path.Type).Method" or "(pkg/path.Type).Field". Setting it
	// replaces the default list.
	Sources []string `json:"sources"`
	// SkipDeclaredAny leaves unchecked assertions on a value read from a
	// declaration this package owns -- a parameter typed any, a field typed
	// any, an element of a container of any -- to the analyzer that reports
	// the declaration.
	SkipDeclaredAny bool `json:"skip-declared-any"`
}

// Default returns the default configuration.
func Default() Config { return Config{Sources: slices.Clone(narrow.DefaultSources)} }

// Analyzer is the analyzer with default configuration.
var Analyzer = New(Default())

// New builds an analyzer for cfg. Flags are bound to a copy of cfg so the
// standalone driver can override options.
func New(cfg Config) *analysis.Analyzer {
	c := &cfg
	a := &analysis.Analyzer{
		Name:     Name,
		Doc:      Doc,
		URL:      "https://github.com/sanketsudake/antislop#" + Name,
		Requires: []*analysis.Analyzer{inspect.Analyzer, seams.Analyzer},
		Run: func(pass *analysis.Pass) (any, error) {
			sources, err := narrow.ParseSources(c.Sources)
			if err != nil {
				return nil, fmt.Errorf("%s: invalid sources entry: %w", Name, err)
			}
			run(pass, *c, sources.Matcher(pass), dictatedSet(pass))
			return nil, nil
		},
	}
	a.Flags.BoolVar(&c.SkipDeclaredAny, "skip-declared-any", c.SkipDeclaredAny, "leave unchecked assertions on a parameter, field, or container element typed any declared in this package to the analyzer that reports the declaration")
	a.Flags.Var(flagx.NewList(&c.Sources), "sources", "comma-separated standard-library sources of untyped values whose immediate result may be asserted without SAFETY (replaces the default list)")
	return a
}

const (
	assertionAdvice = "unchecked type assertion has no SAFETY: justification; state the checked invariant in a preceding \"// SAFETY:\" comment, or use the comma-ok form"
	unsafeAdvice    = "unsafe conversion has no SAFETY: justification; state the invariant that makes it valid in a preceding \"// SAFETY:\" comment"
	linknameAdvice  = "go:linkname has no SAFETY: justification; state the invariant that makes it valid in a preceding \"// SAFETY:\" comment"
)

func run(pass *analysis.Pass, cfg Config, sources *narrow.SourceMatcher, dictated *seams.Set) {
	safety := comments.Safety{}
	generated := passutil.GeneratedFiles{}
	filter := []ast.Node{(*ast.TypeAssertExpr)(nil), (*ast.CallExpr)(nil)}
	passutil.Inspector(pass).WithStack(filter, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push || generated.Skip(stack) {
			return true
		}
		advice := ""
		switch node := n.(type) {
		case *ast.TypeAssertExpr:
			kind, operand, ok := narrow.Site(stack)
			if ok && kind == narrow.Unchecked && !exempt(pass, cfg, sources, dictated, operand, stack) {
				advice = assertionAdvice
			}
		case *ast.CallExpr:
			if isUnsafe(pass, node) {
				advice = unsafeAdvice
			}
		}
		if advice != "" && !safety.Has(pass, n, stack) {
			pass.Reportf(n.Pos(), "%s: %s", Name, advice)
		}
		return true
	})
	for _, file := range passutil.Files(pass) {
		for _, directive := range comments.LinknameDirectives(file) {
			if !safety.HasForDirective(pass, file, directive) {
				pass.Reportf(directive.Pos(), "%s: %s", Name, linknameAdvice)
			}
		}
	}
}

// unsafeCalls are the unsafe functions that reinterpret memory. The rest of
// the package (Sizeof, Alignof, Offsetof) answers questions at compile time.
var unsafeCalls = map[string]bool{"Add": true, "Slice": true, "SliceData": true, "String": true, "StringData": true}

// isUnsafe reports whether call converts to or from unsafe.Pointer, or calls
// one of the unsafe functions that reinterprets memory.
func isUnsafe(pass *analysis.Pass, call *ast.CallExpr) bool {
	fun := ast.Unparen(call.Fun)
	if tv, known := pass.TypesInfo.Types[fun]; known && tv.IsType() {
		if len(call.Args) != 1 {
			return false
		}
		return isPointerType(tv.Type) || isPointerType(pass.TypesInfo.TypeOf(call.Args[0]))
	}
	sel, isSelector := fun.(*ast.SelectorExpr)
	if !isSelector {
		return false
	}
	obj := pass.TypesInfo.Uses[sel.Sel]
	return obj != nil && obj.Pkg() == types.Unsafe && unsafeCalls[obj.Name()]
}

// isPointerType reports whether t is unsafe.Pointer, or a named type over it.
func isPointerType(t types.Type) bool {
	if t == nil {
		return false
	}
	basic, isBasic := t.Underlying().(*types.Basic)
	return isBasic && basic.Kind() == types.UnsafePointer
}

// exempt reports whether an unchecked assertion on operand needs no SAFETY
// comment: the operand is the immediate result of a listed source, or, with
// skip-declared-any, a parameter typed any that noanyparams already reports.
func exempt(pass *analysis.Pass, cfg Config, sources *narrow.SourceMatcher, dictated *seams.Set, operand ast.Expr, stack []ast.Node) bool {
	if sources.Produces(operand, stack) || narrow.IsDictatedParam(pass, dictated, operand, stack) {
		return true
	}
	return cfg.SkipDeclaredAny && narrow.IsDeclaredAny(pass, operand)
}

func dictatedSet(pass *analysis.Pass) *seams.Set {
	// SAFETY: seams.Analyzer declares ResultType *seams.Set and the driver
	// guarantees ResultOf holds a value of that type for every Requires entry.
	return pass.ResultOf[seams.Analyzer].(*seams.Set)
}
