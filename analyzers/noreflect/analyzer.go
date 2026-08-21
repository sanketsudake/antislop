// Package noreflect reports dynamic access and invocation through package reflect.
package noreflect

import (
	"go/ast"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sanketsudake/antislop/internal/flagx"
	"github.com/sanketsudake/antislop/internal/passutil"
)

// Name is the analyzer name and diagnostic prefix.
const Name = "noreflect"

// Doc describes the analyzer.
const Doc = `reports dynamic access and invocation through package reflect

reflect turns a name into a run-time lookup and a value into an untyped
reflect.Value: the compiler stops checking that the field exists, that the
method exists, or that the arguments fit the signature. The evidence the rule
asks for is a typed accessor, a small interface with the methods that are
used, or a decode at the I/O boundary into a struct that names the fields.

Reported: calls to the reflect.Value and reflect.Type methods listed in the
methods option (default Call, CallSlice, FieldByName, FieldByNameFunc,
MethodByName, MapIndex), which look a member up by string or invoke a function
whose signature nothing states. With strict set, every selector into package
reflect is reported as well, including type positions such as reflect.Type.

Not reported: access the compiler can still see the shape of
(reflect.Value.Field, reflect.Value.Index), reflect helpers that take and
return typed values (reflect.DeepEqual, reflect.TypeOf(x).Kind()) unless
strict is set, and methods of the same name declared on types outside package
reflect.

Hashing and equality are the reflect uses with the newest typed replacement:
from Go 1.27, hash/maphash.Hasher[T] states a hash-and-equality contract as an
interface, and maphash.ComparableHasher[T] implements it for any comparable T
without reflect. Instantiating it as ComparableHasher[any] puts the erasure
straight back, so name the element type.

Setting methods replaces the default list rather than adding to it.`

// Config holds the analyzer options.
type Config struct {
	// Methods lists the reflect.Value and reflect.Type method names to report.
	// Setting it replaces the default list.
	Methods []string `json:"methods"`
	// Strict also reports every selector into package reflect.
	Strict bool `json:"strict"`
}

// Default returns the default configuration.
func Default() Config {
	return Config{Methods: []string{"Call", "CallSlice", "FieldByName", "FieldByNameFunc", "MethodByName", "MapIndex"}}
}

// Analyzer is the analyzer with default configuration and flag bindings.
var Analyzer = New(Default())

// New builds an analyzer for cfg. Flags are bound to a copy of cfg so the
// standalone driver can override options.
func New(cfg Config) *analysis.Analyzer {
	c := &cfg
	a := &analysis.Analyzer{
		Name:     Name,
		Doc:      Doc,
		URL:      "https://github.com/sanketsudake/antislop#" + Name,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      func(pass *analysis.Pass) (any, error) { run(pass, *c); return nil, nil },
	}
	a.Flags.Var(flagx.NewList(&c.Methods), "methods", "comma-separated reflect.Value and reflect.Type method names to report (replaces the default list)")
	a.Flags.BoolVar(&c.Strict, "strict", c.Strict, "also report every selector into package reflect")
	return a
}

func run(pass *analysis.Pass, cfg Config) {
	generated := passutil.GeneratedFiles{}
	passutil.Inspector(pass).WithStack([]ast.Node{(*ast.SelectorExpr)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push || generated.Skip(stack) {
			return true
		}
		sel, isSel := n.(*ast.SelectorExpr)
		if !isSel {
			return true
		}
		if cfg.Strict && isReflectPackage(pass, sel.X) {
			pass.Reportf(sel.Pos(), "%s: use of package reflect (reflect.%s) bypasses static types; name the concrete types the code works on, or the small interface it needs",
				Name, sel.Sel.Name)
		}
		recv, ok := reflectMethod(pass, sel)
		if !ok || !slices.Contains(cfg.Methods, sel.Sel.Name) {
			return true
		}
		pass.Reportf(sel.Sel.Pos(), "%s: reflect.%s.%s %s", Name, recv, sel.Sel.Name, advice(sel.Sel.Name))
		return true
	})
}

// isReflectPackage reports whether expr is the package name of an import of
// package reflect, as in the "reflect" of reflect.DeepEqual.
func isReflectPackage(pass *analysis.Pass, expr ast.Expr) bool {
	ident, isIdent := ast.Unparen(expr).(*ast.Ident)
	if !isIdent {
		return false
	}
	name, isPkg := pass.TypesInfo.Uses[ident].(*types.PkgName)
	return isPkg && name.Imported().Path() == "reflect"
}

// reflectMethod returns the name of the reflect type whose method sel selects
// ("Value" or "Type"), or ok == false when sel selects something else. A
// method of the same name on a type of our own is an ordinary method: the
// compiler checks it, so it establishes evidence rather than destroying it.
func reflectMethod(pass *analysis.Pass, sel *ast.SelectorExpr) (recv string, ok bool) {
	selection := pass.TypesInfo.Selections[sel]
	if selection == nil || selection.Kind() != types.MethodVal {
		return "", false
	}
	fn, isFunc := selection.Obj().(*types.Func)
	if !isFunc || fn.Pkg() == nil || fn.Pkg().Path() != "reflect" {
		return "", false
	}
	sig := fn.Signature()
	if sig.Recv() == nil {
		return "", false
	}
	t := sig.Recv().Type()
	if ptr, isPtr := types.Unalias(t).(*types.Pointer); isPtr {
		t = ptr.Elem()
	}
	named, isNamed := types.Unalias(t).(*types.Named)
	if !isNamed {
		return "", false
	}
	return named.Obj().Name(), true
}

// advice states what evidence the call throws away. Call and CallSlice invoke
// a function; the rest look a member up by name.
func advice(method string) string {
	if method == "Call" || method == "CallSlice" {
		return "invokes a function with no compile-time evidence about its signature; call it through a typed func value or interface"
	}
	return "reads a field by string with no compile-time evidence; use a typed accessor or decode into a struct at the boundary"
}
