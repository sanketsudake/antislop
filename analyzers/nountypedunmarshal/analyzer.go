// Package nountypedunmarshal reports decoding into untyped targets.
package nountypedunmarshal

import (
	"go/ast"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sanketsudake/antislop/internal/callspec"
	"github.com/sanketsudake/antislop/internal/flagx"
	"github.com/sanketsudake/antislop/internal/passutil"
	"github.com/sanketsudake/antislop/internal/typesx"
)

// Name is the analyzer name and diagnostic prefix.
const Name = "nountypedunmarshal"

// Doc describes the analyzer.
const Doc = `reports decoding into untyped targets (any, map[string]any, []any)

Decoding is the one moment where a document becomes a Go value, and it is the
cheapest place to say what the document contains. A target typed any,
map[string]any, or []any spends that moment and buys nothing: every later
reader has to assert, index, and guess, and no compiler check stands between
the document and the code. Decode into a struct that names the fields the
program actually uses.

Reported: a call to a decoding function listed in the functions option whose
decode-target argument, after one pointer level is stripped, is the empty
interface, a map whose value type is the empty interface, or a slice whose
element type is the empty interface. A named type declared in the package
under analysis is resolved to its underlying type, so "type Doc
map[string]any" is reported at the call.

Not reported: a struct target, a target whose type defers decoding on purpose
(json.RawMessage, map[string]json.RawMessage), a named type from another
package (text/template.FuncMap is that package's contract), a pass-through
argument typed any rather than *any (it holds whatever pointer the caller
passed; the parameter itself is noanyparams' report), encoding calls such as
json.Marshal, and any function that is not in the list.

Each entry of functions names one function and the index of its decode-target
argument: "pkg/path.Func#N" for a package-level function and
"(*pkg/path.Type).Method#N" (or "(pkg/path.Type).Method#N") for a method.
Setting functions replaces the default list rather than adding to it; a
malformed entry is reported as an analysis error rather than ignored.`

// Config holds the analyzer options.
type Config struct {
	// Functions lists the decoding functions and the index of the decode
	// target in each one. Setting it replaces the default list.
	Functions []string `json:"functions"`
}

// Default returns the default configuration.
func Default() Config {
	return Config{Functions: []string{
		"encoding/json.Unmarshal#1",
		"(*encoding/json.Decoder).Decode#0",
		"encoding/json/v2.Unmarshal#1",
		"encoding/json/v2.UnmarshalRead#1",
		"encoding/json/v2.UnmarshalDecode#1",
		"gopkg.in/yaml.v3.Unmarshal#1",
		"(*gopkg.in/yaml.v3.Decoder).Decode#0",
		"sigs.k8s.io/yaml.Unmarshal#1",
		"github.com/goccy/go-yaml.Unmarshal#1",
		"github.com/BurntSushi/toml.Unmarshal#1",
		"github.com/pelletier/go-toml/v2.Unmarshal#1",
		"github.com/mitchellh/mapstructure.Decode#1",
		"github.com/go-viper/mapstructure/v2.Decode#1",
	}}
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
	}
	a.Run = func(pass *analysis.Pass) (any, error) {
		specs, err := parseSpecs(c.Functions)
		if err != nil {
			return nil, err
		}
		run(pass, specs)
		return nil, nil
	}
	a.Flags.Var(flagx.NewList(&c.Functions), "functions", "comma-separated decoding functions as pkg/path.Func#N or (*pkg/path.Type).Method#N (replaces the default list)")
	return a
}

// spec names one decoding function and the position of its decode target.
type spec struct {
	callspec.Spec
	target int // index of the decode-target argument
}

// parseSpecs decodes the functions option. It runs on every pass rather than
// at construction so a malformed entry -- which reaches us from a host's
// settings file -- surfaces as an analysis error instead of a panic.
func parseSpecs(entries []string) ([]spec, error) {
	out := make([]spec, 0, len(entries))
	for _, entry := range entries {
		text, target, err := callspec.SplitTarget(entry)
		if err != nil {
			return nil, &syntaxError{entry: entry}
		}
		cs, err := callspec.Parse(text)
		if err != nil {
			return nil, &syntaxError{entry: entry}
		}
		out = append(out, spec{Spec: cs, target: target})
	}
	return out, nil
}

// syntaxError explains a malformed functions entry.
type syntaxError struct{ entry string }

func (e *syntaxError) Error() string {
	return Name + ": invalid functions entry " + strconv.Quote(e.entry) +
		`; write "pkg/path.Func#N" or "(*pkg/path.Type).Method#N", where N is the index of the decode-target argument`
}

func run(pass *analysis.Pass, specs []spec) {
	generated := passutil.GeneratedFiles{}
	passutil.Inspector(pass).WithStack([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push || generated.Skip(stack) {
			return true
		}
		call, isCall := n.(*ast.CallExpr)
		if !isCall || call.Ellipsis.IsValid() {
			return true
		}
		fn, recv, found := callspec.Callee(pass, call)
		if !found {
			return true
		}
		for _, s := range specs {
			if !s.Matches(fn, recv) || s.target >= len(call.Args) {
				continue
			}
			arg := call.Args[s.target]
			target, untyped := untypedTarget(pass, pass.TypesInfo.TypeOf(arg))
			if !untyped {
				continue
			}
			pass.Reportf(arg.Pos(), "%s: %s into %s keeps the document untyped; decode into a struct that names the fields you use",
				Name, callspec.Display(fn, recv), target)
			return true
		}
		return true
	})
}

// untypedTarget reports whether the decode target keeps the document untyped,
// and renders it for the diagnostic. One pointer level is stripped
// first: the target reaches the decoder as &v or as a pointer variable.
func untypedTarget(pass *analysis.Pass, t types.Type) (rendered string, untyped bool) {
	if t == nil || !declaredHere(t, pass.Pkg) {
		return "", false
	}
	// Only a pointer is a decode target. A bare any argument statically holds
	// whatever pointer the caller passed; that signature is noanyparams' report.
	ptr, isPtr := t.Underlying().(*types.Pointer)
	if !isPtr {
		return "", false
	}
	t = ptr.Elem()
	if typesx.IsEmptyInterface(t) {
		return "any", true
	}
	if !declaredHere(t, pass.Pkg) {
		return "", false
	}
	switch under := t.Underlying().(type) {
	case *types.Map:
		if typesx.IsEmptyInterface(under.Elem()) {
			return "map[" + types.TypeString(under.Key(), types.RelativeTo(pass.Pkg)) + "]any", true
		}
	case *types.Slice:
		if typesx.IsEmptyInterface(under.Elem()) {
			return "[]any", true
		}
	}
	return "", false
}

// declaredHere reports whether t is written out at the call site or named by a
// type this package declares. A named type from another package states that
// package's contract, so its shape is not this call's decision.
func declaredHere(t types.Type, pkg *types.Package) bool {
	switch named := t.(type) {
	case *types.Alias:
		return named.Obj().Pkg() == nil || named.Obj().Pkg() == pkg
	case *types.Named:
		return named.Obj().Pkg() == nil || named.Obj().Pkg() == pkg
	}
	return true
}
