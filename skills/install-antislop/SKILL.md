---
name: install-antislop
description: Wires the antislop Go linter (go/analysis analyzers plus a golangci-lint module plugin) into a Go repository and its CI. Trigger it when a user asks for antislop, for opinionated low-evidence lint rules in Go, or for these analyzers under golangci-lint.
---

# Install antislop

Wire the antislop analyzers into the current Go repository and into the lint setup it already has.
antislop rejects code that destroys or fabricates type evidence:
`any` in signatures, fields and containers,
narrowing back out of `any`,
`reflect`, monkey patching, structural names, and untyped decoding.
Every rule is on and every finding is an error; there is no autofix.
An existing codebase adopts through a baseline ratchet or `new-from-rev`, not by disabling rules.

Leave work that is not yours untouched, and follow the configuration style the repository already uses.

## Procedure

1. Learn the repository before you change anything in it:
   - Read the agent instructions it ships (`AGENTS.md`, `CLAUDE.md`, `.cursorrules`).
   - Run `git status`; whatever is already modified stays as it is.
   - Read the `go` directive in `go.mod`.
     antislop needs a Go 1.26 or newer toolchain to build the custom golangci-lint binary.
     A repository that targets an older Go version can still be linted;
     only the toolchain that builds the binary has to be current.
   - Find the golangci-lint configuration: `.golangci.yml`, `.golangci.yaml`, or `.golangci.toml`.
   - Look for an existing `.custom-gcl.yml`, for other custom linters, and for an antislop entry.
     Read the diff before you replace any of them.

2. Resolve the current versions rather than trusting remembered ones:
   - `go list -m -versions github.com/sanketsudake/antislop` for the published tags; take the last one.
   - `golangci-lint --version` for the version the repository already uses.
   - antislop pins `golang.org/x/tools` to the version golangci-lint v2.12.x pins,
     because `golangci-lint custom` builds both into one binary.
     If the repository is on a different golangci-lint minor version, say so,
     and prefer the standalone path (Path B) over a version mismatch that fails to build.

3. Configure the linter. Take the path that matches the repository.

   **Path A — golangci-lint is already in use.**
   - Create or merge `.custom-gcl.yml` from `assets/custom-gcl.yml`:
     `version` is the repository's golangci-lint version,
     and the plugin entry carries `module`, `import`, and the resolved antislop `version`.
     `import` is required: the `register.Plugin` call lives in the `plugin/` subpackage,
     so the module path alone registers nothing.
     If the file already lists plugins, add an entry; do not replace the list.
   - Merge into `.golangci.yml`, keeping every existing linter, setting, and exclusion:
     `antislop` in `linters.enable`,
     and `linters.settings.custom.antislop` with `type: module`,
     a `description`, and the settings block from `assets/golangci-settings.yml`.
     That asset lists every analyzer and every option at its default, one comment per option;
     keep only the entries you actually change.
   - Run `golangci-lint custom` to build `./custom-gcl`, and add `custom-gcl` to `.gitignore`.
   - CI must build the binary too, and run `./custom-gcl run ./...` instead of `golangci-lint run`:
     merge the job in `assets/github-actions.yml` into the repository's lint workflow.
     A plain `golangci-lint run` does not know about antislop and silently skips it.

   **Path B — no golangci-lint.**
   - Pin the analyzer with a tool directive and invoke it with `go tool`:
     `go get -tool github.com/sanketsudake/antislop/cmd/antislop`, then
     `go tool antislop ./...` from the Makefile or CI lint step
     (`assets/Makefile.snippet`).
     A tool directive pins the version in `go.mod`, so the lint result does not
     depend on the day it ran and a dependency bot can see it.
   - **Do not use `go run`.** It reports its own exit 1 for any non-zero child
     status, so "found findings" (3) and "failed to run" (1) become the same
     answer, and it writes progress lines into the output stream. Anything that
     gates on the exit code or parses the output breaks silently. An installed
     binary on `$PATH` is equally fine.
   - Exit codes: 0 no findings, 3 findings, 1 an error, 2 a bad flag.
   - Disable an analyzer with `-<name>=false` and set an option with `-<name>.<option>=<value>`;
     `antislop -help` lists them.

   **Test files (both paths).** antislop lints `_test.go` files by default.
   If the user wants production code only, use the host's mechanism rather than an analyzer option:
   `-test=false` for the standalone binary, or a `linters.exclusions.rules` entry
   with `path: _test\.go` and `linters: [antislop]` under golangci-lint.
   Ask before adding it; it is a policy choice, and record it with a comment in the config.

4. Run the linter and read what it says. On a repository with existing code,
   run `antislop -summary ./...` first
   and show the user the counts per analyzer and package before enabling anything.
   For rollout, prefer a ratchet over disabling analyzers:
   golangci-lint's `issues.new-from-rev` (new code only), or, on the standalone path,
   a baseline — `antislop -baseline <file> -update ./...` to record what is there today and
   `antislop -baseline <file> ./...` to gate, which fails on any file/analyzer pair that is
   missing or grows and passes one that shrinks. Check the file in; each line is a claim a
   reviewer can challenge.
   Where only one package legitimately uses a pattern, scope the rule instead of dropping it:
   `-<name>.exclude 'pkg/router/...'` (or golangci-lint's `issues.exclude-rules`) keeps the rule
   guarding every other package.
   Use the volume knobs (`noanycontainers.encoders`, `nonarrowany.sources`, `skip-declared-any`,
   `-test=false`) before `disable`; record every deviation with a comment in the config.
   Report what the analyzers find in the repository's own source.
   Change that source only if the user asked for a migration or a cleanup.

   Do not make lint pass by suppressing rules, disabling analyzers, adding `//nolint`,
   or laundering a type through `any(x)`.
   The intended remedies are semantic:
   - decode input once at its I/O boundary into a named type, and pass that type on;
   - use the comma-ok form instead of an unchecked assertion;
   - write a `// SAFETY:` comment that states a real, checked invariant, not a restatement of the line;
   - inject a dependency through a small interface instead of patching a package-level variable.

5. Review the final diff and report:
   - files changed,
   - versions installed (antislop and golangci-lint),
   - configuration merged,
   - checks run,
   - remaining findings, and what each one would take to fix.

6. Migration guidance.
   antislop stays generic and carries no per-repository exceptions,
   so a rule that applies to one project only goes into a linter of its own.
   Use `settings.disable` only with a written reason in a comment next to the entry,
   so the next reader knows what was traded away and why.
   When replacing an older antislop setup, compare its settings before overwriting:
   an option that was changed on purpose is a decision, not drift.

## Assets

| File | Use |
|---|---|
| `assets/custom-gcl.yml` | `.custom-gcl.yml` for the module plugin build (Path A). |
| `assets/golangci-settings.yml` | Every analyzer and option at its default, for `linters.settings.custom.antislop` (Path A). Generated from the registry. |
| `assets/github-actions.yml` | CI job that builds `custom-gcl` and runs it (Path A). |
| `assets/Makefile.snippet` | Standalone `go run` lint target (Path B). |
