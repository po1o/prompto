# Spec: Daemon Mode Cleanup & Documentation

Status: DRAFT — pending human approval before Phase 2 (Plan).
Last updated: 2026-05-23.

## Objective

The daemon-mode subsystem of `prompto` is **operational but ugly**:
internally complex, inconsistently named, sparsely documented, and unevenly
tested. This spec scopes a single coordinated effort to:

1. **Simplify** the daemon's internals (split, rename, dedupe, redraw module
   boundaries) without changing any externally observable behavior.
2. **Document** the daemon end-to-end so a new contributor can read one page
   and understand the lifecycle, the Registry / Hard-vs-Soft-Cancel model, and
   the per-shell integration contract.
3. **Cover** the refactored code with Go tests where coverage is currently
   missing, so future cleanup can proceed safely.

**Target users:**
- prompto contributors who need to extend the daemon (new RPC, new
  shell integration, new cancellation case).
- Shell-integration maintainers (Zsh / Fish / PWSH / Bash+ble.sh) who need to
  reason about the `--repaint` contract.
- Future agents (LLMs) reading this codebase cold.

**Non-objectives:**
- No changes to the gRPC wire protocol (`src/daemon/ipc/daemon.proto`).
- No changes to user-visible CLI flags (`prompto render`, `--repaint`,
  daemon subcommands).
- No changes to shell-visible behavior (prompt latency, repaint semantics,
  vim-mode handling all stay identical).
- No new features. Pre-existing TODOs / FIXMEs may be addressed only when
  removing them strictly simplifies the surrounding code.

## Tech Stack

- **Language:** Go (`go 1.26.0`, module `github.com/po1o/prompto/src`).
- **RPC:** gRPC + Protocol Buffers (`src/daemon/ipc/`, generated via
  `go generate ./...`).
- **Shells:** Zsh, Fish, PowerShell (PSReadLine), Bash via `ble.sh`. Shell
  scripts are embedded Go assets under `src/shell/scripts/prompto.{bash,fish,ps1,zsh}`.
- **Test framework:** standard `testing` + `github.com/alecthomas/assert`
  (already in `go.mod`). Use existing helpers; do not introduce new test
  libraries.
- **Lint stack** (already enforced in CI, see `.github/workflows/code.yml`):
  `golangci-lint`, `fieldalignment`, `modernize`.

## Commands

All commands run from `/Users/polo/development/prompto/src/`.

```sh
# Regenerate protobuf / gRPC stubs (run if .proto changes — must produce no diff in CI)
go generate ./...

# Build the binary
go build ./...

# Run the full Go test suite (CI uses -count=1 -v)
go test -count=1 ./...

# Run only the daemon package tests (fast iteration)
go test -count=1 ./daemon/...

# Lint (matches CI)
golangci-lint run

# Struct field alignment (CI enforces; ipc/ package is exempt)
go list ./... | grep -v '/daemon/ipc$' | xargs fieldalignment

# Modernize check (CI enforces; ipc/ package is exempt)
go list ./... | grep -v '/daemon/ipc$' | xargs modernize
```

A change is "green" only when **all** of the above pass with no diff and no
warnings.

## Project Structure

In-scope directories (this spec touches these):

```
src/daemon/                  ← Daemon core: lifecycle, RPC server, render
                               pipeline, session runtime, registry, cancellation,
                               cache, watchers, IPC socket helpers. ~6.2k LoC.
src/daemon/ipc/              ← Generated gRPC code + proto. EDIT proto only;
                               regenerate stubs via `go generate`.
src/cli/daemon.go            ← `prompto daemon …` subcommands (start/stop/etc).
src/cli/daemon_unix.go       ← Unix-specific daemon CLI plumbing.
src/cli/daemon_windows.go    ← Windows-specific daemon CLI plumbing.
src/shell/init.go            ← Shell init dispatch — picks the right embedded script.
src/shell/{bash,fish,pwsh,zsh}.go         ← Per-shell render-invocation helpers.
src/shell/scripts/prompto.{bash,fish,ps1,zsh}  ← Embedded shell scripts: where
                               `--repaint`, vim-mode hooks, and daemon integration
                               actually live.
src/prompt/         src/config/        src/segments/      src/template/
src/cache/          src/runtime/
```

Out-of-scope directories (do not touch unless a thin boundary change is
unavoidable, and even then surface it for approval first):

```
src/themes/        themes/
```

Documentation targets:

```
src/daemon/ARCHITECTURE.md           ← Expanded / rewritten. Canonical daemon
                                       architecture doc. Subsumes both
                                       .claude/docs/*-vim-mode-plan.md.
docs/maintainers/daemon-shells.md    ← NEW. Per-shell integration & debugging
                                       guide (Zsh / Fish / PWSH / Bash+ble.sh).
                                       Lives under maintainers/ to keep
                                       public docs/ user-facing.
.claude/docs/daemon-vim-mode-plan.md ← Mark as HISTORICAL (header pointer to
.claude/docs/shell-vim-mode-plan.md    src/daemon/ARCHITECTURE.md). Keep for
                                       provenance; do not delete.
```

## Code Style

Follow existing repo conventions (visible across `src/daemon/*.go`). Concretely:

```go
// Package daemon runs the long-lived render service. The daemon exposes a
// gRPC API (see ipc/daemon.proto), maintains per-session render runtimes,
// and reuses in-flight segment computations across "repaint" requests
// (see ARCHITECTURE.md, "Hard vs. Soft Cancellation").
package daemon

// Registry tracks in-flight segment computations so that a repaint request
// can subscribe to an already-running computation instead of restarting it.
//
// Cancellation policy:
//   - Hard cancel (new command): registry context is cancelled; in-flight
//     futures abort and MUST NOT write to the cache.
//   - Soft cancel (vim toggle): registry context is preserved; in-flight
//     futures continue and their results are reused.
type Registry struct {
    mu      sync.Mutex
    entries map[SegmentCacheKey]*future
}

// Subscribe returns the future for key, creating one (and starting compute
// via start) if absent. The returned future is safe for concurrent receive.
func (r *Registry) Subscribe(key SegmentCacheKey, start func(context.Context) Result) *future {
    // ...
}
```

Conventions:

- **Package comment** at the top of one file per package, summarizing what the
  package does and where to look next.
- **Exported symbol comments** are mandatory; they explain WHY/contract, not
  WHAT (the signature already says what).
- **Internal comments** only where the code surprises a careful reader
  (invariants, race-condition reasoning, cancellation edges).
- **Naming:** verbs for funcs (`Subscribe`, `Cancel`, `StartRender`), nouns
  for types. Avoid `Manager`, `Helper`, `Util` suffixes — name the thing it
  *does*, not its role.
- **Files:** one concept per file. Prefer `registry.go` + `registry_test.go`
  over `daemon_helpers.go`.
- **Errors:** wrap with `fmt.Errorf("daemon: …: %w", err)` so logs are
  greppable.
- **Concurrency:** every shared mutable field documents what guards it
  (`// guarded by mu`). Prefer channels + small structs over giant mutex
  scopes.
- **Line length:** ≤180 cols (matches `.golangci.yml`).
- **Lint clean:** every change passes `golangci-lint`, `fieldalignment`, and
  `modernize` (see Commands).

## Testing Strategy

- **Framework:** standard `testing` + `github.com/alecthomas/assert`. Test
  files live next to source (`registry.go` → `registry_test.go`).
- **Levels:**
  - **Unit tests** for every public type/function in `src/daemon/` whose
    behavior is non-trivial (Registry subscribe/cancel, ReloadGate, RequestManager,
    RenderPipeline branches, SessionStore lifecycle).
  - **Integration tests** in `src/daemon/server_test.go` /
    `service_test.go` covering the full RPC path for the canonical scenarios
    in `.claude/docs/daemon-vim-mode-plan.md`:
      - Scenario 1: Soft cancel during vim toggle (registry hit, fast path).
      - Scenario 2: Hard cancel on new command (registry aborts, cache safe).
      - Scenario 3: Rapid-fire toggles (computation runs once).
  - **Shell-script tests** stay in `src/shell/daemon_scripts_test.go`
    (Go-driven snapshot/regex checks of the embedded scripts).
- **Coverage target:** every file refactored under this spec ends with
  meaningful test coverage of its branches. Specific % is not the goal;
  "every cancellation path and every error path has at least one test" is.
- **Concurrency tests:** anything touching the Registry or cancellation MUST
  run under `go test -race ./daemon/...` cleanly. CI does not currently run
  `-race`; we run it locally before declaring a refactor done.
- **No mocking of the database/RPC wire** in integration tests — use real
  in-process gRPC (`bufconn` or equivalent) the existing tests use.
- **Determinism:** no `time.Sleep`-based assertions. Use channels, contexts,
  or fake clocks.

## Boundaries

**Always do:**
- Run `go test -count=1 ./daemon/...` and `golangci-lint run` before every
  commit; run `go test -race ./daemon/...` before declaring a task done.
- Keep the gRPC wire (`daemon.proto`), CLI flags, and shell-observable
  behavior identical to `main` at every commit boundary.
- Update `src/daemon/ARCHITECTURE.md` in the SAME commit that changes the
  architecture it describes.
- For every renamed/moved symbol: verify all call sites with `grep`, not just
  the compiler — generated code (`daemon.pb.go`) and shell scripts may
  reference Go-exported names via string literals.
- Land changes incrementally: one subsystem per PR (Registry, RenderPipeline,
  SessionStore, etc.), each independently revertible.

**Ask first:**
- Any change to `src/daemon/ipc/daemon.proto` (even comment-only edits trigger
  a regen + CI gate).
- Any change to public Go API outside `src/daemon/` (the prompt engine, CLI
  surface, segment interfaces).
- Any new dependency in `go.mod`.
- Any change to CI config (`.github/workflows/`).
- Splitting a single subsystem across more than one PR.
- Deleting tests, even ones that look redundant.
- Renaming files that are referenced by other docs or by `.claude/docs/`.

**Never do:**
- Change the wire protocol or CLI flag surface "while you're in there."
- Add `// TODO` / `// FIXME` without an issue link.
- Skip `go generate` and let regenerated stubs drift.
- Introduce a new abstraction (interface, factory, manager) without
  removing at least one existing one. Net abstraction count must not
  increase in this effort.
- Mock the cache or RPC in integration tests.
- Add `time.Sleep` to make a flaky test pass.
- Commit with `--no-verify` or skip lint with `//nolint` unless the
  surrounding line is annotated with WHY.
- Delete `.claude/docs/*-vim-mode-plan.md` — mark historical and link
  forward to the canonical doc.

## Success Criteria

This spec is "done" when ALL of the following hold:

1. **Tests green** — `go test -count=1 ./...`, `golangci-lint run`,
   `fieldalignment`, `modernize`, and `go test -race ./daemon/...` all pass
   from a clean checkout.
2. **Wire unchanged** — `git diff main -- src/daemon/ipc/daemon.proto` is
   empty. `git diff main -- src/cli/daemon*.go` shows no new/removed flags
   (renames-only OK).
3. **Behavior unchanged** — the three canonical scenarios from the vim-mode
   plan (Soft cancel, Hard cancel, Rapid toggles) have integration tests
   that pass on `main` AND on this branch with identical assertions.
4. **Coverage gained** — every file under `src/daemon/` modified in this
   effort either has new test coverage for its cancellation/error paths or
   already had complete coverage at the start.
5. **Docs landed:**
   - `src/daemon/ARCHITECTURE.md` is rewritten as the canonical daemon doc,
     covering: gRPC entry → Service → RenderPipeline → SessionRuntime →
     Registry → Cache, with the Hard/Soft Cancel model explicit and the
     three vim-mode scenarios diagrammed.
   - `docs/maintainers/daemon-shells.md` exists and covers, per shell:
     mode-detection mechanism, redraw trigger, `--repaint` wiring, ble.sh
     dependency note for Bash, and debugging tips.
   - Both `.claude/docs/*-vim-mode-plan.md` carry a "HISTORICAL — see
     `src/daemon/ARCHITECTURE.md`" banner.
   - Every exported symbol in `src/daemon/*.go` has a doc comment.
6. **Subsystem boundaries cleaner** — at least the following are obviously
   improved (judgment call, but visible in diff): Registry / RenderPipeline
   separation, RequestManager naming and responsibilities, SessionStore
   ownership, ReloadGate vs ConfigWatcher boundary. Net file count in
   `src/daemon/` should not increase materially; net LoC should drop or
   stay flat.

## Open Questions

1. **Public docs surfacing:** should `docs/maintainers/daemon-shells.md`
   instead live under `src/daemon/` (closer to code) or under `docs/`
   (discoverable on the website)? Default: `docs/maintainers/` per
   assumption #4 above.
2. **`fieldalignment` scope:** the IPC package is already exempt. Are there
   other generated/oddly-shaped files we should exempt during this cleanup,
   or do we just fix the alignment as we touch each file?
3. **Race detector in CI:** worth adding `go test -race ./daemon/...` to
   `.github/workflows/code.yml` as part of this work, or punt to a follow-up?
   (Listed under "Ask first: CI config".)
4. **Subsystem PR order:** any preferred order (e.g., docs-first then code,
   or code-first then docs)? Default plan: write the new
   `src/daemon/ARCHITECTURE.md` first (as the agreed target shape), then
   refactor subsystem-by-subsystem to match it.
5. **Soft-deprecation of `.claude/docs/*-vim-mode-plan.md`:** keep them in
   the repo as historical, or move under `.claude/docs/archive/`?
