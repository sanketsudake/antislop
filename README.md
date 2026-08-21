# antislop

Opinionated `go/analysis` analyzers that reject low-evidence Go patterns.

## Philosophy

- **Parse at the boundary.**
  Decode a document, a request, or an environment once, where it enters the program, into a named type;
  everything downstream works on that type.
- **Never fabricate evidence.**
  An assertion, a `reflect` call, or a monkey patch claims a fact the compiler cannot see.
  If the fact is real, encode it in a type; if it is not, the code is already wrong.
- **Preserve inference.**
  A value that arrives with a known concrete type keeps it.
  Widening to `any` and asserting back is a round trip that can only lose information.
- **Name for the domain role, not the structure.**
  `Order` says what a value is; `OrderShape` says only that it has fields.
- **Justify escape hatches.**
  An unchecked assertion, an `unsafe` conversion, or a `go:linkname` you cannot avoid
  carries a `// SAFETY:` comment stating the invariant that makes it sound.
- **Test through real seams.**
  Inject a dependency through a small interface;
  do not reassign a package-level function variable from a test.

## Install

### Standalone

```bash
go get -tool github.com/sanketsudake/antislop/cmd/antislop
go tool antislop ./...
```

Prefer `go tool` (or an installed binary) over `go run`.
`go run` reports its own exit 1 for any non-zero child status,
so "found findings" (3) becomes indistinguishable from "failed to run" (1),
and it writes progress lines into the output stream —
both matter to anything that gates on the result.

Exit codes: **0** no findings, **3** findings reported, **1** an error (bad packages, unreadable baseline), **2** a bad flag.

Disable an analyzer with `-<name>=false`; set an option with `-<name>.<option>=<value>` (`antislop -help` lists them).
Skip paths with `-exclude` (every analyzer) or `-<name>.exclude` (one analyzer); see [Excluding paths](#excluding-paths).

On an existing codebase, start with the summary instead of the finding list:

```bash
antislop -summary ./...          # counts per analyzer and per package
antislop -summary -test=false ./...
```

It answers "which analyzers, and where" before you decide what to enable;
the same `-<name>=false` and `-<name>.<option>` flags apply.

### golangci-lint (module plugin)

`.custom-gcl.yml`:

```yaml
version: v2.13.0
plugins:
  - module: github.com/sanketsudake/antislop
    import: github.com/sanketsudake/antislop/plugin
    version: v0.2.0
```

`import` is required:
the `register.Plugin` call lives in the `plugin/` subpackage,
so the module path alone registers nothing.

`.golangci.yml`:

```yaml
version: "2"
linters:
  enable:
    - antislop
  settings:
    custom:
      antislop:
        type: module
        description: Rejects low-evidence Go patterns
        settings:
          disable: []
          noanyparams:
            allow-variadic: true
```

Then build the binary and run it:

```bash
golangci-lint custom
./custom-gcl run -c .golangci.yml ./...
```

Requires Go 1.26 or newer to build the custom binary.
Build it with Go 1.27 to analyse code that uses Go 1.27 language features.

### Toolchain version

Build antislop with a Go toolchain at least as new as the code it analyses.
The analyzers type-check your packages with the `go/types` of the toolchain that built the binary,
so a 1.26-built binary cannot parse a generic method or a promoted-field struct literal.
It does not silently pass them:
the package fails to load, every analyzer reports `analysis skipped due to errors in package`,
and the driver exits **1**, not 0 —
so a build that gates on the exit code fails rather than reporting a clean run.

antislop's own module stays on `go 1.26.0`, so 1.26 remains supported;
only the toolchain you build with has to keep up with the code you point it at.

### Agent skill

`skills/install-antislop/SKILL.md` wires the configuration above into a repository; point your coding agent at it.

## Rules

<!-- rules:start -->
| Analyzer | Rejects |
|---|---|
| [`noanyparams`](docs/rules.md#noanyparams) | parameters typed as the empty interface |
| [`noanyreturns`](docs/rules.md#noanyreturns) | function results typed as the empty interface |
| [`noanytypes`](docs/rules.md#noanytypes) | type declarations that merely rename the empty interface |
| [`noanyfields`](docs/rules.md#noanyfields) | struct fields typed as the empty interface |
| [`noanycontainers`](docs/rules.md#noanycontainers) | maps (and optionally slices, arrays and channels) whose element or key type is the empty interface |
| [`nonarrowany`](docs/rules.md#nonarrowany) | checked narrowing of an empty-interface value (comma-ok assertions and type switches) |
| [`safetycomment`](docs/rules.md#safetycomment) | unchecked escape hatches with no SAFETY comment: single-value type assertions, unsafe conversions, and go:linkname |
| [`nochainedassert`](docs/rules.md#nochainedassert) | assertions chained through the empty interface (x.(any).(T), any(x).(T)) |
| [`noknownwidening`](docs/rules.md#noknownwidening) | known concrete values stored into empty-interface locations |
| [`nowidenassert`](docs/rules.md#nowidenassert) | values widened to the empty interface and later asserted back in the same function |
| [`noreflect`](docs/rules.md#noreflect) | dynamic access and invocation through package reflect |
| [`nomonkeypatch`](docs/rules.md#nomonkeypatch) | monkey patching: patch libraries and test-time reassignment of package-level function variables |
| [`nountypedunmarshal`](docs/rules.md#nountypedunmarshal) | decoding into untyped targets (any, map[string]any, []any) |
| [`nostructuralnames`](docs/rules.md#nostructuralnames) | identifiers that contain a forbidden structural term (default: shape) |
<!-- rules:end -->

Full descriptions, options, and the reasoning behind each rule: [docs/rules.md](docs/rules.md).

## Violation examples

Each snippet is rejected by the named analyzer; the accepted counterpart is in [`example/`](example/).

### `noanyparams`

```go
func Save(value any) {}
```

### `noanyreturns`

```go
func Load() any { return nil }
```

### `noanytypes`

```go
type Metadata = any
```

### `noanyfields`

```go
type Event struct {
	Payload any
}
```

### `noanycontainers`

```go
type Document map[string]any
```

### `nonarrowany`

```go
var raw any

func RawPort() (int, bool) { port, ok := raw.(int); return port, ok }
```

### `safetycomment`

```go
func FirstName(values []fmt.Stringer) string {
	return values[0].(interface{ Name() string }).Name()
}
```

### `nochainedassert`

```go
func Count(v int) int {
	// SAFETY: v was widened to any on the line itself, so it is still an int.
	return any(v).(int)
}
```

Walking a decoded document one assertion at a time is the same mistake, spelled longer
(also reported by `noanycontainers` for each `map[string]any`):

```go
return payload.(map[string]any)["viewport"].(map[string]any)["width"].(int)
```

### `noknownwidening`

```go
var retries any = 3
```

### `nowidenassert`

The declaration is also reported by `noknownwidening`, and the comma-ok test by `nonarrowany`:
one mistake, seen by three rules.

```go
func Double(n int) int {
	var boxed any = n
	if v, ok := boxed.(int); ok {
		return v * 2
	}
	return 0
}
```

### `noreflect`

```go
func ID(u User) string { return reflect.ValueOf(u).FieldByName("ID").String() }
```

### `nomonkeypatch`

```go
func TestClock(t *testing.T) {
	clock = func() time.Time { return time.Time{} }
}
```

### `nountypedunmarshal`

The variable is also reported by `noanycontainers`: the map is untyped where it is
declared and again where it is filled.

```go
func ParseSettings(b []byte) error {
	var settings map[string]any
	return json.Unmarshal(b, &settings)
}
```

### `nostructuralnames`

```go
type OrderShape struct{ ID string }
```

## Suppressing a finding

Under golangci-lint, `//nolint:antislop` is the only mechanism,
and it suppresses every antislop analyzer on that line — read the rule before you write it.
The standalone binary has no suppression comment at all:
turn an analyzer off with `-<name>=false` for the whole run, in the open, where a reviewer sees it.

A `// SAFETY:` comment is not a suppression.
It is the documentation `safetycomment` requires:
state the invariant that makes the escape hatch sound,
not the fact that the line exists.

## Adopting on an existing codebase

There are two ways to adopt without either a mass edit or a disabled rule set.
Both keep every rule on; they differ in what they gate against.

**A baseline**, when you run the standalone binary:

```bash
antislop -baseline .antislop-baseline -update ./...   # record what is there today
antislop -baseline .antislop-baseline ./...           # gate: fails on anything new
```

The file records a count per file and analyzer.
A pair that is missing, or that grows, fails; a pair that shrinks passes and can be re-recorded.
It is a ratchet, not an amnesty — the tree can only get cleaner, and every new finding is still an error.

It carries no line numbers on purpose.
Keying on them churns the file on every edit above a finding,
which is how a baseline stops being regenerated and starts being ignored.
The cost is that a net-zero swap inside one file and analyzer is not caught; diff stability is worth more.
Check the file in, and treat each line as a claim about that file that a reviewer can challenge.

**`new-from-rev`**, when you run under golangci-lint:
`issues.new-from-rev: <sha>` (or `new: true`) reports only findings introduced after a revision,
so the whole rule set can be on from day one.

Either way:

1. Measure first: `antislop -summary ./...` (add `-test=false` to see production code alone).
   The per-package table tells you where the untyped surface is.
2. Turn the volume rules down, not off, where the domain is genuinely untyped
   (a JSON-document engine, a protocol bridge):
   `noanycontainers.encoders`, `nonarrowany.sources`, `skip-declared-any`, and `-test=false` are the knobs;
   `disable` is the last resort and deserves a comment in the config.
3. Prefer a path exclusion over disabling a rule module-wide when only one package is affected —
   see [Excluding paths](#excluding-paths).
4. Never make lint pass by laundering types (`any(x)`, `//nolint` without a reason, a `SAFETY:` that states no invariant).

## Excluding paths

Under golangci-lint, scope a linter with `issues.exclude-rules` as usual.
The standalone binary is its own host, so it carries the same capability:

```bash
antislop -exclude 'pkg/gen/...' ./...                      # every analyzer skips that subtree
antislop -nostructuralnames.exclude 'pkg/router/...' ./...  # one analyzer skips it
```

Patterns match the path relative to the directory the run started in, with forward slashes:

| Pattern | Matches |
|---|---|
| `pkg/gen/...` | that directory and everything below it |
| `pkg/a/a.go` | exactly that file |
| `*_test.go` | that base name at any depth (a pattern with no `/` matches the base name) |
| `pkg/*.go` | one segment only — `*` does not cross a slash |

Reach for the per-analyzer form first.
A rule that is wrong for one package is usually right for the rest,
and `-<name>.exclude` keeps it guarding them;
disabling the rule module-wide gives that up to fix one place.
Exclusion is a driver concern — the rules themselves never learn which project is running them.

## Test files

antislop analyzes `_test.go` files like any other Go source.
On a real project most findings land in tests
(table rows typed `[]any`, rendered YAML held as `map[string]any`, `v.(T)` on test data),
so many teams choose to lint production code first.
Both frontends make that a one-line policy, in the open:

- Standalone: `antislop -test=false ./...` (the multichecker's `-test` flag skips test files).
- golangci-lint: an exclusion rule scoped to antislop —

  ```yaml
  linters:
    exclusions:
      rules:
        - path: _test\.go
          linters: [antislop]
  ```

Exempting tests is a policy choice, not a rule default:
the analyzers themselves have no test-file switch.

## Development

```bash
make check   # lint, test, dogfood, smoke, deps-check, gendocs-check
```

`analyzers/` is canonical: one package per rule with `analysistest` fixtures.
`example/` holds one file per rule with the README violation and its accepted counterpart;
`example/expected.txt` is the golden diagnostic list (`make smoke-update` after intentional changes).

Fixtures that need a language feature newer than the module's `go` directive
live in a `go127`-style testdata package behind a `//go:build` tag,
with the same tag on the test that drives them,
so the older CI leg keeps compiling.

[`docs/go1.27.md`](docs/go1.27.md) records what Go 1.27 changed for these rules,
and which candidate rules were considered and rejected.

## Credits

The set of rules follows [dmmulroy/anti-slop](https://github.com/dmmulroy/anti-slop) (MIT)
and its thesis that code must not destroy or fabricate type evidence.

## License

MIT.
