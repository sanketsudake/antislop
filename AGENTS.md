# Repository guidance

- `analyzers/` holds one package per rule; `antislop.go` is the registry.
  Adding a rule = new package + registry entry + `example/` snippet + README row.
- A rule must hold for any Go repository.
  Nothing in `analyzers/` names a product, a company, or one project's import path;
  a check that needs an exception for a single codebase belongs in that codebase's own linter.
- Use `go/analysis` with `pass.TypesInfo`; do not add another parser or type checker.
- Every rule ships `analysistest` fixtures with `// want` comments: at least one valid and one invalid case per behaviour bullet in its `Doc`.
- Diagnostics are prescriptive: say what evidence is missing and how to establish it.
  No autofixes; fixes are semantic and belong to the human.
- Pin `golang.org/x/tools` to the version the targeted golangci-lint release pins (`make deps-check`).
- Run `make check` before committing.
