// Package nomonkeypatch reports monkey patching: patch libraries and
// test-time reassignment of package-level function variables.
package nomonkeypatch

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/sanketsudake/antislop/internal/flagx"
	"github.com/sanketsudake/antislop/internal/passutil"
)

// Name is the analyzer name and diagnostic prefix.
const Name = "nomonkeypatch"

// Doc describes the analyzer.
const Doc = `reports monkey patching: patch libraries and test-time reassignment of package-level function variables

A patch library rewrites a function while the program runs, and a test that
reassigns a package-level func variable does the same thing by hand. Either
way the call site keeps naming one function while another one runs, and
nothing in the types records the substitution -- so the test proves something
about a program that is not the one that ships. Pass the dependency in
instead: a constructor argument, a parameter, or a struct field the test fills
in states the substitution in the type.

Reported: an import of any package listed in the packages option (default
bou.ke/monkey, github.com/agiledragon/gomonkey,
github.com/agiledragon/gomonkey/v2, github.com/undefinedlabs/go-mpatch), in
every file including tests; and, unless allow-package-var-stubbing is set, an
assignment in a _test.go file whose left-hand side names a package-level
variable of a function type.

Not reported: the same assignment outside a test, which is ordinary package
initialisation; local function variables, which belong to one test; struct
fields (srv.now = ...), which are the seam this rule asks for; short variable
declarations, which introduce a new name rather than replacing a call; and
package-level variables that are not function typed.

Setting packages replaces the default list rather than adding to it.`

// Config holds the analyzer options.
type Config struct {
	// Packages lists the import paths of patch libraries.
	// Setting it replaces the default list.
	Packages []string `json:"packages"`
	// AllowPackageVarStubbing permits a test to reassign a package-level
	// function variable, the hand-written form of patching.
	AllowPackageVarStubbing bool `json:"allow-package-var-stubbing"`
}

// Default returns the default configuration.
func Default() Config {
	return Config{Packages: []string{
		"bou.ke/monkey",
		"github.com/agiledragon/gomonkey",
		"github.com/agiledragon/gomonkey/v2",
		"github.com/undefinedlabs/go-mpatch",
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
		Run:      func(pass *analysis.Pass) (any, error) { run(pass, *c); return nil, nil },
	}
	a.Flags.Var(flagx.NewList(&c.Packages), "packages", "comma-separated import paths of patch libraries to report (replaces the default list)")
	a.Flags.BoolVar(&c.AllowPackageVarStubbing, "allow-package-var-stubbing", c.AllowPackageVarStubbing,
		"allow a test to reassign a package-level function variable")
	return a
}

func run(pass *analysis.Pass, cfg Config) {
	reportPatchImports(pass, cfg)
	if cfg.AllowPackageVarStubbing {
		return
	}
	reportPackageVarStubs(pass)
}

// reportPatchImports reports every import of a listed patch library. The
// import is the decision; the individual Patch calls only carry it out.
func reportPatchImports(pass *analysis.Pass, cfg Config) {
	for _, file := range passutil.Files(pass) {
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil || !slices.Contains(cfg.Packages, path) {
				continue
			}
			pass.Reportf(spec.Pos(), "%s: import of %s patches functions at runtime; inject the dependency through an interface or a function parameter instead",
				Name, path)
		}
	}
}

// reportPackageVarStubs reports a test that replaces a package-level function
// variable, which is patching written by hand.
func reportPackageVarStubs(pass *analysis.Pass) {
	generated := passutil.GeneratedFiles{}
	passutil.Inspector(pass).WithStack([]ast.Node{(*ast.AssignStmt)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push || generated.Skip(stack) {
			return true
		}
		assign, isAssign := n.(*ast.AssignStmt)
		if !isAssign || assign.Tok != token.ASSIGN || !passutil.IsTestFile(pass, assign.Pos()) {
			return true
		}
		for _, lhs := range assign.Lhs {
			if name, ok := packageFuncVar(pass, lhs); ok {
				pass.Reportf(name.Pos(), "%s: test reassigns package-level function variable %q; pass the dependency in through the constructor or a parameter instead of patching the package",
					Name, name.Name)
			}
		}
		return true
	})
}

// packageFuncVar reports whether lhs names a package-level variable of a
// function type. A selector (srv.now) is a field, which is the recommended
// seam, and a local variable belongs to the test that declared it.
func packageFuncVar(pass *analysis.Pass, lhs ast.Expr) (*ast.Ident, bool) {
	name, isIdent := ast.Unparen(lhs).(*ast.Ident)
	if !isIdent || name.Name == "_" {
		return nil, false
	}
	v, isVar := pass.TypesInfo.Uses[name].(*types.Var)
	if !isVar || v.Parent() != pass.Pkg.Scope() {
		return nil, false
	}
	if _, isSignature := v.Type().Underlying().(*types.Signature); !isSignature {
		return nil, false
	}
	return name, true
}
