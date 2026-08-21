# Go 1.27 review

What Go 1.27 changed for these rules, and what was deliberately left alone.
The short version:
1.27 adds no new low-evidence pattern,
so it adds no new rule —
it adds new ways to *keep* evidence,
which is advice, not detection.

## Compatibility

antislop's module stays on `go 1.26.0`.
1.26 is still supported and still tested in CI.

The toolchain you *build* with is a separate matter.
The analyzers type-check your packages with the `go/types` of the toolchain that produced the binary,
so a 1.26-built binary cannot parse a generic method or a promoted-field struct literal.
It fails loudly rather than quietly:
the package does not load,
every analyzer reports `analysis skipped due to errors in package`,
and the driver exits **1**.

Fixtures that need 1.27 live in a `go127` testdata package behind `//go:build go1.27`,
with a matching build tag on the test that drives them.
The tag does double duty:
it keeps the file out of a 1.26 build,
and it sets the file's language version,
which is what the version gate reads.

## Rules that changed

### `noanyparams`, `noanyreturns`

A method declaration may now declare its own type parameters.
Before 1.27, a method that had to be polymorphic in its argument or result had no in-namespace option
and `any` was a defensible answer;
now the type parameter keeps the caller's type through the call instead of erasing it.

On a **concrete method** in a file compiled at 1.27 or newer,
the diagnostic names that alternative alongside the named type.
It is gated three ways:

- **Language version.**
  Read from the file's own version, falling back to the package's.
  An unknown version reports false,
  so the rule never proposes a fix the analyzed code cannot compile.
  A 1.26 module gets the old wording for identical source.
- **Concrete methods only.**
  Interface methods may not declare type parameters,
  and a generic method may not implement one,
  so an interface method keeps the plain advice.
  Plain functions keep it too — generic functions long predate 1.27.
- **Not already generic.**
  A method that already declares type parameters keeps the plain advice:
  the `any` is a separate decision.

### `nomonkeypatch`

`httptest.NewTestServer(t, handler)` serves over an in-memory network,
so an injected `http.Client` or `RoundTripper` needs no port and no patched transport.
`testing/synctest` with `synctest.Sleep` does the same for a patched `time.Now`.
Both are the real seams this rule already asks for, named in its `Doc`.

### `noreflect`

`hash/maphash.Hasher[T]` states a hash-and-equality contract as an interface,
and `maphash.ComparableHasher[T]` implements it for any comparable `T` without reflect.
Noted in the `Doc`, with the caveat that `ComparableHasher[any]` puts the erasure straight back.

### `nountypedunmarshal`

`encoding/json/v2` is generally available rather than a `GOEXPERIMENT`;
its decoders were already on the default list and stay there.
Stricter v2 defaults do not change this rule's reading:
a target typed `any` erases the document either way.
`jsontext.Value` defers a decode the way `json.RawMessage` does —
`[]byte`, not `any` — and is now pinned as exempt by a fixture
rather than passing only through the "named type from another package" clause.

## Considered and rejected

- **A new rule.**
  Nothing in 1.27 creates a pattern that destroys evidence.
  Generic methods, `strings.CutLast`, `uuid`, `simd`, and `math/big.Int.Divide`
  all add typed APIs.
- **`database/sql.ConvertAssign#0` on `nountypedunmarshal`.**
  A conversion helper, not a document boundary.
  Consistent with not covering `Rows.Scan`.
- **A rule for `encoding/json/v2` strictness**
  (rejecting duplicate keys, invalid UTF-8).
  Real, but a correctness setting rather than an evidence question,
  and v2 already defaults to it.
- **The unbuffered-timer-channel change** and the removal of `asynctimerchan`.
  Runtime behaviour, not a typing decision.

## Struct literal keys: what actually shipped

The release note says a struct literal key "may now be any valid field selector".
The compiler accepts the **promoted field name as a plain identifier** —
`Outer{Payload: 42}` — and rejects the dotted form:

```go
Outer{Mid.Payload: 42}   // invalid field name Mid.Payload in struct literal
```

So `KeyValueExpr.Key` stays an `*ast.Ident` and no analyzer needed changing.
The effect is only that a key may now resolve to a field of an *embedded* type.
No rule keys off a literal's field names:
`noanyfields` reports the field's declaration,
`noanycontainers` reports the literal's type,
and `noknownwidening` leaves composite-literal fields to those two by design.
