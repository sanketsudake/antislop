// Package noanycontainers reports containers whose key or element type is the
// empty interface.
package noanycontainers

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sanketsudake/antislop/internal/callspec"
	"github.com/sanketsudake/antislop/internal/flagx"
	"github.com/sanketsudake/antislop/internal/passutil"
	"github.com/sanketsudake/antislop/internal/typesx"
)

// Name is the analyzer name and diagnostic prefix.
const Name = "noanycontainers"

// Doc describes the analyzer.
const Doc = `reports maps (and optionally slices, arrays and channels) whose element or key type is the empty interface

A map[string]any is a document that was never really decoded: its shape stayed
in the wire format and every reader rebuilds it with assertions. Decode at the
I/O boundary into a struct, or into a map of a named type. An any key is the
same loss on the other side: nothing states what identifies an entry.

Reported: a map type expression whose direct key or direct value type is the
empty interface (any / interface{}) or a same-package alias or named type
that resolves to it, in every position: parameters, results, fields,
variables, type declarations, composite literal types, conversions, and make
calls. The outermost offending node of a type expression is reported once, so
map[any]map[string]any yields a single diagnostic.

With slices set, slices, arrays and channels of the empty interface are
reported too, which also makes the inner []any of map[string][]any the
reported node.

Not reported: type parameters (map[string]T), named types from other packages
(map[string]driver.Value, text/template.FuncMap, json.RawMessage); a container
literal (or make call) written directly as the argument of an encoder in the
encoders list (json.Marshal(map[string]any{...}), json.NewEncoder(w).Encode(...)),
which is an output boundary -- the document is serialised on the spot, not
carried around untyped -- setting encoders replaces the default list and an
empty list reports those literals too; with skip-declared-any set, a container
literal (or make call) that fills a slot declared in this package -- returned
from an own function, passed to an own function, assigned to an own field or
typed variable -- because the slot's declaration is reported already and one
decision should yield one finding (a literal that declares its own type, m :=
map[string]any{...}, and a literal passed to another package's function are
still reported); and — unless slices is set — []any, [N]any and channels.
Variadic ...any belongs to noanyparams.`

// Config holds the analyzer options.
type Config struct {
	// Slices extends the rule from maps to slices, arrays and channels.
	Slices bool `json:"slices"`
	// Encoders lists functions whose direct container-literal arguments are
	// an output boundary and are not reported, as "pkg/path.Func" or
	// "(*pkg/path.Type).Method". Setting it replaces the default list.
	Encoders []string `json:"encoders"`
	// SkipDeclaredAny leaves a container literal (or make call) that fills a
	// slot declared in this package -- a result, a parameter of an own
	// function, a field, a typed variable -- to the finding at that
	// declaration: one finding per decision instead of one per use.
	SkipDeclaredAny bool `json:"skip-declared-any"`
}

// DefaultEncoders are the standard and common third-party encoders.
var DefaultEncoders = []string{
	"encoding/json.Marshal",
	"encoding/json.MarshalIndent",
	"(*encoding/json.Encoder).Encode",
	"encoding/json/v2.Marshal",
	"encoding/json/v2.MarshalWrite",
	"encoding/json/v2.MarshalEncode",
	"gopkg.in/yaml.v3.Marshal",
	"(*gopkg.in/yaml.v3.Encoder).Encode",
	"sigs.k8s.io/yaml.Marshal",
	"github.com/goccy/go-yaml.Marshal",
	"(*github.com/BurntSushi/toml.Encoder).Encode",
	"github.com/pelletier/go-toml/v2.Marshal",
}

// Default returns the default configuration.
func Default() Config {
	return Config{Slices: false, Encoders: slices.Clone(DefaultEncoders)}
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
		Run: func(pass *analysis.Pass) (any, error) {
			encoders, err := callspec.ParseAll(c.Encoders)
			if err != nil {
				return nil, fmt.Errorf("%s: invalid encoders entry: %w", Name, err)
			}
			run(pass, *c, encoders)
			return nil, nil
		},
	}
	a.Flags.BoolVar(&c.Slices, "slices", c.Slices, "also report slices, arrays and channels whose element type is any")
	a.Flags.BoolVar(&c.SkipDeclaredAny, "skip-declared-any", c.SkipDeclaredAny, "leave a container literal that fills a slot declared in this package to the finding at that declaration")
	a.Flags.Var(flagx.NewList(&c.Encoders), "encoders", "comma-separated encoders whose direct container-literal arguments are not reported (replaces the default list)")
	return a
}

func run(pass *analysis.Pass, cfg Config, encoders []callspec.Spec) {
	filter := []ast.Node{(*ast.MapType)(nil)}
	if cfg.Slices {
		filter = append(filter, (*ast.ArrayType)(nil), (*ast.ChanType)(nil))
	}
	generated := passutil.GeneratedFiles{}
	passutil.Inspector(pass).WithStack(filter, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push || generated.Skip(stack) {
			return true
		}
		if encodedLiteral(pass, stack, encoders) {
			return false // the whole literal type is an output boundary
		}
		if cfg.SkipDeclaredAny && declaredSlot(pass, stack) {
			return false // reported once, at the declaration of the slot
		}
		// Stop at the first offending node so that a container nested inside a
		// reported one does not produce a second diagnostic for the same type
		// expression. Only the key and element type expressions are pruned;
		// the values of a composite literal are siblings, not children.
		return !check(pass, n)
	})
}

// check reports the container node when its direct key or element type is an
// owned empty interface, and returns whether it reported.
func check(pass *analysis.Pass, n ast.Node) bool {
	switch c := n.(type) {
	case *ast.MapType:
		if erased(pass, c.Key) {
			report(pass, c, "uses an untyped key; use a named key type that states what identifies an entry")
			return true
		}
		if erased(pass, c.Value) {
			report(pass, c, "erases the value types; decode the document at its I/O boundary into a struct (or a map of a named type)")
			return true
		}
	case *ast.ArrayType:
		if erased(pass, c.Elt) {
			kind := "slice"
			if c.Len != nil {
				kind = "array"
			}
			report(pass, c, "erases the element types; use a "+kind+" of a named type")
			return true
		}
	case *ast.ChanType:
		if erased(pass, c.Value) {
			report(pass, c, "erases the element types; use a channel of a named type")
			return true
		}
	}
	return false
}

func report(pass *analysis.Pass, container ast.Expr, advice string) {
	pass.Reportf(container.Pos(), "%s: %s %s", Name, types.ExprString(container), advice)
}

func erased(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	return t != nil && typesx.IsEmptyInterfaceOwnedBy(t, pass.Pkg)
}

// encodedLiteral reports whether the container type at the top of stack is
// the type of a composite literal (or make call) written directly as an
// argument to one of the encoders: json.Marshal(map[string]any{...}),
// json.NewEncoder(w).Encode(&map[string]any{...}). Nested literal types
// inside such an argument are part of the same document.
func encodedLiteral(pass *analysis.Pass, stack []ast.Node, encoders []callspec.Spec) bool {
	if len(encoders) == 0 {
		return false
	}
	// Walk outward: the container type sits under a CompositeLit (its Type),
	// possibly under further composite literals, an optional &, then the call.
	for i := len(stack) - 2; i >= 0; i-- {
		switch parent := stack[i].(type) {
		case *ast.CompositeLit, *ast.KeyValueExpr, *ast.ParenExpr:
			continue
		case *ast.UnaryExpr:
			if parent.Op == token.AND {
				continue
			}
			return false
		case *ast.CallExpr:
			if i == len(stack)-2 {
				// make(map[string]any): the type is the make argument itself.
				if id, ok := ast.Unparen(parent.Fun).(*ast.Ident); !ok || id.Name != "make" || pass.TypesInfo.Uses[id] != types.Universe.Lookup("make") {
					return false
				}
				continue
			}
			fn, owner, ok := callspec.Callee(pass, parent)
			if !ok {
				return false
			}
			for _, e := range encoders {
				if e.Matches(fn, owner) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
	return false
}

// declaredSlot reports whether the container type at the top of stack belongs
// to a composite literal (or make call) that fills a slot declared in this
// package: a return value of an own function, an argument to an own function
// or method, the right-hand side of an assignment to an own variable or
// field, a typed variable declaration, or a field of an own struct literal.
// The slot's declaration carries the finding.
func declaredSlot(pass *analysis.Pass, stack []ast.Node) bool {
	i := len(stack) - 2
	// The container type may sit inside a larger type expression
	// ([]map[string]any); climb to the expression that uses the type.
	for i >= 0 {
		if _, isType := stack[i].(*ast.ArrayType); isType {
			i--
			continue
		}
		if _, isType := stack[i].(*ast.MapType); isType {
			i--
			continue
		}
		break
	}
	if i < 0 {
		return false
	}
	// The container type must be the type of a literal or the argument of make.
	switch parent := stack[i].(type) {
	case *ast.CompositeLit:
	case *ast.CallExpr:
		id, ok := ast.Unparen(parent.Fun).(*ast.Ident)
		if !ok || id.Name != "make" || pass.TypesInfo.Uses[id] != types.Universe.Lookup("make") {
			return false
		}
	default:
		return false
	}
	value := stack[i]
	for i--; i >= 0; i-- {
		switch parent := stack[i].(type) {
		case *ast.ParenExpr:
			continue
		case *ast.UnaryExpr:
			if parent.Op != token.AND {
				return false
			}
			continue
		case *ast.ReturnStmt:
			return true
		case *ast.CallExpr:
			fn, _, ok := callspec.Callee(pass, parent)
			return ok && fn.Pkg() == pass.Pkg
		case *ast.KeyValueExpr:
			continue // judged at the enclosing composite literal
		case *ast.CompositeLit:
			// A literal nested in a container literal is part of the same
			// document: keep walking to the outer literal's slot. A field of
			// an own struct literal is a declared slot.
			t := pass.TypesInfo.TypeOf(parent)
			if isContainerType(t) {
				value = parent
				continue
			}
			return ownedStruct(pass, t)
		case *ast.AssignStmt:
			if parent.Tok != token.ASSIGN {
				return false
			}
			for j, rhs := range parent.Rhs {
				if rhs == value && j < len(parent.Lhs) {
					return ownedLocation(pass, parent.Lhs[j])
				}
			}
			return false
		case *ast.ValueSpec:
			return parent.Type != nil
		default:
			return false
		}
	}
	return false
}

func ownedLocation(pass *analysis.Pass, lhs ast.Expr) bool {
	switch l := ast.Unparen(lhs).(type) {
	case *ast.Ident:
		v, ok := pass.TypesInfo.Uses[l].(*types.Var)
		return ok && v.Pkg() == pass.Pkg
	case *ast.SelectorExpr:
		field, _, ok := callspec.Field(pass, l)
		return ok && field.Pkg() == pass.Pkg
	}
	return false
}

func ownedStruct(pass *analysis.Pass, t types.Type) bool {
	if t == nil {
		return false
	}
	if ptr, isPtr := types.Unalias(t).(*types.Pointer); isPtr {
		t = ptr.Elem()
	}
	named, isNamed := types.Unalias(t).(*types.Named)
	if !isNamed {
		_, isStruct := t.Underlying().(*types.Struct)
		return isStruct // an anonymous struct literal declares its fields here
	}
	return named.Obj().Pkg() == pass.Pkg
}

func isContainerType(t types.Type) bool {
	if t == nil {
		return false
	}
	if ptr, isPtr := types.Unalias(t).(*types.Pointer); isPtr {
		t = ptr.Elem()
	}
	switch t.Underlying().(type) {
	case *types.Map, *types.Slice, *types.Array:
		return true
	}
	return false
}
