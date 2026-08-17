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
go run github.com/sanketsudake/antislop/cmd/antislop@latest ./...
```

Disable an analyzer with `-<name>=false`; set an option with `-<name>.<option>=<value>` (`antislop -help` lists them).

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
version: v2.12.2
plugins:
  - module: github.com/sanketsudake/antislop
    import: github.com/sanketsudake/antislop/plugin
    version: v0.1.0
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

antislop ships no baseline on purpose; the host has better tools for rollout.

1. Measure first: `antislop -summary ./...` (add `-test=false` to see production code alone).
   The per-package table tells you where the untyped surface is.
2. Gate new code, fix old code as you touch it:
   golangci-lint's `issues.new-from-rev: <sha>` (or `new: true`) reports only findings introduced after a revision,
   so the whole rule set can be on from day one without a mass edit.
3. Turn the volume rules down, not off, where the domain is genuinely untyped
   (a JSON-document engine, a protocol bridge):
   `noanycontainers.encoders`, `nonarrowany.sources`, `skip-declared-any`, and `-test=false` are the knobs;
   `disable` is the last resort and deserves a comment in the config.
4. Never make lint pass by laundering types (`any(x)`, `//nolint` without a reason, a `SAFETY:` that states no invariant).

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

## Credits

The set of rules follows [dmmulroy/anti-slop](https://github.com/dmmulroy/anti-slop) (MIT)
and its thesis that code must not destroy or fabricate type evidence.

## License

MIT.
