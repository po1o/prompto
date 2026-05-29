# Tasks: Daemon Mode Cleanup & Documentation

Status: DRAFT — pending human approval before Phase 4 (Implement).
References: [SPEC.md](./SPEC.md) · [PLAN.md](./PLAN.md).
Last updated: 2026-05-24.

## How To Read This File

- Tasks are **strictly dependency-ordered**. Do not start task N until N-1 is merged unless explicitly marked parallel.
- Each task is sized to ~5 files of net change (tests counted generously — a refactor that touches `foo.go` + `foo_test.go` counts as 1 file pair).
- "Verify" commands all run from `/Users/polo/development/prompto/src/`.
- "Spec link" points back to the SPEC section the task advances; cite it in the PR description.
- `[P]` = can run in parallel with sibling `[P]` tasks at the same depth.

Universal acceptance gates (apply to EVERY task unless overridden):

- `go test -count=1 ./...` green.
- `golangci-lint run` clean.
- `fieldalignment` and `modernize` clean (per CI's grep-excluded `ipc/` rule).
- `go test -race ./daemon/...` green.
- `git diff main -- src/daemon/ipc/daemon.proto` empty unless the task explicitly edits the proto.
- `git diff main -- src/cli/daemon*.go` shows no new/removed CLI flags (renames OK).

---

## Phase A — Audit + Target Shape

### A1. Daemon current-state map

- **Acceptance:** `.claude/notes/daemon-current-state.md` exists with: one-line "what this file does today" for every file in components 1–11 (per PLAN.md table), plus a "callers crossing layer boundaries" subsection listing every cross-layer call discovered.
- **Verify:** `wc -l .claude/notes/daemon-current-state.md` is non-empty; reviewer reads it and can answer "what does `coordinator.go` do today?" without opening the file.
- **Files:** `.claude/notes/daemon-current-state.md` (new).
- **Spec link:** SPEC Project Structure; PLAN Phase A1.

### A2. Target `src/daemon/ARCHITECTURE.md` (draft)

- **Acceptance:** Rewritten ARCHITECTURE.md describes the end-state, not today's state. Must contain: (a) layered diagram (gRPC → Server → Service → RenderPipeline → SessionRuntime → Registry → Cache), (b) `CancelKind` type as a first-class concept, (c) the three vim-mode scenarios as numbered walkthroughs, (d) explicit list of "what each file owns" matching the post-refactor layout.
- **Verify:** Human review. Subsequent Phase C PRs must move code toward this doc; deviations require updating the doc in the same PR.
- **Files:** `src/daemon/ARCHITECTURE.md`.
- **Spec link:** SPEC Success Criteria #5; PLAN Phase A2.
- **Decision needed:** ASCII vs mermaid diagrams (PLAN Q3). Default: ASCII unless `.github/` or `docs/` already uses mermaid.

### A3. `docs/maintainers/daemon-shells.md` skeleton [P with A2]

- **Acceptance:** New file with one section per shell (Zsh / Fish / PWSH / Bash+ble.sh). Each section has the four headings: **Detection mechanism**, **Redraw trigger**, **`--repaint` wiring**, **Debugging tips**. Bodies may be stubs; headings and structure must be final.
- **Verify:** Reviewer can scan it and confirm the contract surface is captured even with empty bodies.
- **Files:** `docs/maintainers/daemon-shells.md` (new), possibly `docs/maintainers/README.md` (new, one paragraph explaining what maintainers/ is for).
- **Spec link:** SPEC Success Criteria #5; PLAN Phase A3.

### A — Phase Gate

- Human reviews A1+A2+A3 artifacts. Approve before B1 starts.

---

## Phase B — Test Baseline

### B1. Daemon coverage audit

- **Acceptance:** `.claude/notes/daemon-coverage.md` lists per-package `go test -cover` percentage and per-file branch gaps for components 1–11. Highlights every cancellation path and every error path that is currently uncovered.
- **Verify:** `go test -cover ./daemon/... ./cli/... ./shell/...` produces numbers matching the doc. Reviewer can grep "uncovered" to see the gap list.
- **Files:** `.claude/notes/daemon-coverage.md` (new).
- **Spec link:** SPEC Testing Strategy; PLAN Phase B1.

### B2. Characterization tests for the three vim-mode scenarios

- **Acceptance:** `src/daemon/scenarios_test.go` exists with three integration tests using in-process gRPC (existing `bufconn` setup): `TestScenario_SoftCancel_VimToggle`, `TestScenario_HardCancel_NewCommand`, `TestScenario_RapidFireToggles_RunsOnce`. Each asserts the cache-pollution / registry-reuse / single-execution invariant from `.claude/docs/daemon-vim-mode-plan.md`.
- **Verify:** `go test -count=1 -race -run TestScenario_ ./daemon/` green on `main` (the goal is to lock current behavior, not to drive new behavior).
- **Files:** `src/daemon/scenarios_test.go` (new). Possibly tiny test helpers in `src/daemon/testutil_test.go` (new) if existing helpers don't suffice.
- **Spec link:** SPEC Testing Strategy bullet "Integration tests"; PLAN Phase B2.

### B3. Race-detector baseline

- **Acceptance:** `go test -race -count=3 ./daemon/...` runs three consecutive clean passes on `main` (or `cleanup-base` branch). Any flake or known race is documented in `.claude/notes/daemon-race-baseline.md` with reproducer and (separately, not in this task) filed as an issue. Note: this task documents the baseline; it does not fix races.
- **Verify:** Three back-to-back clean runs captured.
- **Files:** `.claude/notes/daemon-race-baseline.md` (new).
- **Spec link:** SPEC Testing Strategy bullet "Concurrency tests"; PLAN Phase B3.
- **Decision needed:** PLAN Q4 — add `-race` to CI as part of this task or later?

### B — Phase Gate

- Characterization tests merged. Race baseline documented. No code changes to daemon proper yet. Approve before C1 starts.

---

## Phase C — Daemon Core Refactor

Each task = one PR. Sequential unless `[P]` marker present.

### C1a. Introduce `CancelKind` and `RegistryContext` types (additive)

- **Acceptance:** New types defined in `src/daemon/registry.go`: `CancelKind` (enum: `CancelHard`, `CancelSoft`), `RegistryContext` (wraps context.Context with cancel-kind awareness). No existing callers migrated yet — purely additive. Doc comments explain the Hard/Soft model per `src/daemon/ARCHITECTURE.md`.
- **Verify:** Universal gates + `go vet ./daemon/...`. Scenarios tests from B2 still green.
- **Files:** `src/daemon/registry.go`, `src/daemon/registry_test.go` (add tests for new types).
- **Spec link:** SPEC Code Style example block; PLAN Phase C1.

### C1b. Migrate Registry internals to `CancelKind`

- **Acceptance:** `Registry.Subscribe`, `Registry.Cancel`, and the internal `future` struct use `CancelKind`. External signatures unchanged where possible; if a signature must change, only the daemon-internal callers are touched in this PR.
- **Verify:** Universal gates + scenarios tests green. No call sites outside `src/daemon/` changed.
- **Files:** `src/daemon/registry.go`, `src/daemon/registry_test.go`.
- **Spec link:** PLAN Phase C1.

### C1c. Migrate `request_manager.go` to `CancelKind`

- **Acceptance:** `RequestManager` uses `CancelKind` to choose Hard vs Soft on cancel. The "is this a repaint?" check is now expressed as `CancelKind`, not a bool flag.
- **Verify:** Universal gates + scenarios tests green. `git grep -n "repaint" src/daemon/` shows the new naming consistently.
- **Files:** `src/daemon/request_manager.go`, `src/daemon/request_manager_test.go`, `src/daemon/registry.go` (only if interface tightening required).
- **Spec link:** PLAN Phase C1.

### C1d. Migrate `update_hub.go`, `update_binding.go`, `stream_relay.go` to `CancelKind`

- **Acceptance:** All three files honor `CancelKind` on cancellation paths. No `bool isRepaint` parameter survives. Cache writes are gated behind `ctx.Err() == nil` per the daemon-vim-mode-plan "Cache Safety" rule.
- **Verify:** Universal gates + scenarios tests green. `git grep -n "isRepaint\|is_repaint" src/daemon/` returns nothing.
- **Files:** `src/daemon/update_hub.go`, `src/daemon/update_binding.go`, `src/daemon/stream_relay.go`, plus their `_test.go` siblings.
- **Spec link:** PLAN Phase C1.

### C2a. Decide and document Service/RenderPipeline/Coordinator boundaries

- **Acceptance:** Update `src/daemon/ARCHITECTURE.md` "Core Components" section with the chosen post-refactor boundaries. Decision options written into the PR description: (a) absorb Coordinator into Service, (b) absorb into RenderPipeline, (c) keep three but rename. One chosen, justified, and approved by human before C2b.
- **Verify:** Human approval on the PR. No code changes yet.
- **Files:** `src/daemon/ARCHITECTURE.md`.
- **Spec link:** PLAN Phase C2.

### C2b. Execute Service/RenderPipeline/Coordinator merge per C2a

- **Acceptance:** Code matches the boundary decision from C2a. Net file count in `src/daemon/` does not increase; net LoC stays flat or drops.
- **Verify:** Universal gates + scenarios tests green. `wc -l src/daemon/*.go | tail -1` ≤ baseline.
- **Files:** `src/daemon/service.go`, `src/daemon/render_pipeline.go`, `src/daemon/coordinator.go` (likely deleted or merged), plus their `_test.go` siblings.
- **Spec link:** PLAN Phase C2.

### C3a. Extract OS-detection into `src/daemon/sessionid/` subpackage

- **Acceptance:** Files `session_freebsd*.go`, `session_linux.go`, `session_macos.go`, `session_other.go`, `session_windows.go` move under `src/daemon/sessionid/` with a single exported `Resolve() (string, error)` function. `src/daemon/session.go` calls `sessionid.Resolve()`.
- **Verify:** Universal gates green on all platforms (CI matrix already covers ubuntu/macos/windows). `go build ./...` clean.
- **Files:** `src/daemon/sessionid/*.go` (new subpackage), `src/daemon/session.go` (call site), `src/daemon/session_test.go` (adjust). Old `session_{os}.go` files deleted.
- **Spec link:** PLAN Phase C3.

### C3b. Clarify Session vs SessionStore vs Runtime boundaries

- **Acceptance:** Doc comments on each of `Session`, `SessionStore`, `SessionRenderRuntime` explicitly state "owns X, does NOT own Y." Naming aligned with ARCHITECTURE.md. Any field that crosses the boundary either moves or is justified in a comment.
- **Verify:** Universal gates + scenarios tests green. Reviewer reads the three doc comments and can answer "where does field Z live?" without grep.
- **Files:** `src/daemon/session.go`, `src/daemon/session_store.go`, `src/daemon/runtime.go`, plus `_test.go` siblings.
- **Spec link:** PLAN Phase C3.

### C4a. Split `server.go` by RPC method group

- **Acceptance:** `src/daemon/server.go` (currently 560 LoC) split into `server.go` (wiring/lifecycle) + one file per RPC method group (e.g., `server_render.go`, `server_cache.go`, `server_session.go`). No method count change.
- **Verify:** Universal gates green. `wc -l src/daemon/server*.go` shows the new layout. `go doc ./daemon | grep -c '^func.*Server'` matches the count on `main`.
- **Files:** `src/daemon/server.go` + 2–3 new `server_*.go` files + `src/daemon/server_test.go` (split if natural).
- **Spec link:** PLAN Phase C4.

### C4b. Align `src/cli/daemon*.go` to new server shape

- **Acceptance:** CLI plumbing files reflect any naming changes from C4a. No new flags, no removed flags.
- **Verify:** Universal gates green. `prompto daemon --help` output matches `main`'s output verbatim (capture both, diff).
- **Files:** `src/cli/daemon.go`, `src/cli/daemon_unix.go`, `src/cli/daemon_windows.go`.
- **Spec link:** PLAN Phase C4.

### C5. Split `client.go`  `[P with C6, C7, C8]`

- **Acceptance:** `src/daemon/client.go` (336 LoC) split into `client.go` (public surface), `client_dial.go` (connect/retry), `client_rpc.go` (RPC wrappers). Same external API.
- **Verify:** Universal gates green. Tests pass. No callers outside `src/daemon/` need changes.
- **Files:** 3 client files + `client_test.go` (split if natural).
- **Spec link:** PLAN Phase C5.

### C6a. Resolve ReloadGate vs ConfigWatcher overlap  `[P with C5, C7, C8]`

- **Acceptance:** One of: (a) ReloadGate absorbed into ConfigWatcher, (b) explicit doc comments stating "ReloadGate handles X, ConfigWatcher handles Y" if both must stay. ARCHITECTURE.md updated accordingly.
- **Verify:** Universal gates + scenarios tests green. Reviewer can answer "who debounces config reloads?" by reading either file's package comment.
- **Files:** `src/daemon/reload_gate.go`, `src/daemon/config_watcher.go`, plus `_test.go` siblings.
- **Spec link:** PLAN Phase C6.

### C6b. Collapse `lock_*.go` trio if possible  `[P with C5, C7, C8]`

- **Acceptance:** `lock.go` + `lock_unix.go` + `lock_windows.go` (260 LoC total) audited. Either: (a) collapsed to `lock.go` if the platform split was unnecessary, (b) doc comments justify the split if it's load-bearing.
- **Verify:** Universal gates green on the CI matrix (ubuntu/macos/windows). `go build ./...` clean per platform.
- **Files:** `src/daemon/lock.go`, `src/daemon/lock_unix.go`, `src/daemon/lock_windows.go`.
- **Spec link:** PLAN Phase C6.

### C7. Resolve `prompt_cache_bridge.go` indirection  `[P with C5, C6, C8]`

- **Acceptance:** Either: (a) the bridge is removed and `cache.go` calls into `src/cache/` directly, (b) the bridge is renamed to describe its actual job, with a doc comment explaining the impedance mismatch it resolves.
- **Verify:** Universal gates + scenarios tests green.
- **Files:** `src/daemon/cache.go`, `src/daemon/prompt_cache_bridge.go`, `src/daemon/cache_test.go`.
- **Spec link:** PLAN Phase C7.

### C8. Add proto doc comments + environment audit  `[P with C5, C6, C7]`

- **Acceptance:** Every message and RPC in `src/daemon/ipc/daemon.proto` has a `//` comment explaining purpose and cancel semantics where relevant. `go generate ./...` produces a diff that is committed. `src/daemon/environment.go` reviewed and either left as-is (with a one-line "audit OK" PR note) or trivially cleaned up.
- **Verify:** Universal gates green. `git diff` after `go generate` is empty post-commit.
- **Files:** `src/daemon/ipc/daemon.proto`, `src/daemon/ipc/daemon.pb.go` (regenerated), `src/daemon/ipc/daemon_grpc.pb.go` (regenerated), `src/daemon/environment.go`.
- **Spec link:** PLAN Phase C8.

### C — Phase Gate

- All C tasks merged. Daemon proper is clean. Human go/no-go for Phase D before any of D1–D4 are scheduled.

---

## Phase E — Shell Integration

Runs in parallel with the Phase D gate decision; E does not depend on D.

### E1. Audit + align `src/shell/{init,bash,fish,pwsh,zsh}.go`

- **Acceptance:** Doc comments on each per-shell helper describe the `--repaint` wire-up. Duplication across the four files surfaced — either deduped into `shell/init.go` or justified per-shell in a comment.
- **Verify:** Universal gates green. `go test ./shell/...` green.
- **Files:** `src/shell/init.go`, `src/shell/bash.go`, `src/shell/fish.go`, `src/shell/pwsh.go`, `src/shell/zsh.go`.
- **Spec link:** PLAN Phase E1.

### E2. Embedded script header comments + snapshot tests

- **Acceptance:** Each of `prompto.{bash,fish,ps1,zsh}` has a top-of-file comment block explaining: (a) mode-detection mechanism for this shell, (b) where `--repaint` is appended, (c) any ble.sh dependency. `daemon_scripts_test.go` has a snapshot test per shell that locks current behavior — any future drift fails CI loudly.
- **Verify:** Universal gates green. `go test ./shell -run TestDaemonScripts` green.
- **Files:** `src/shell/scripts/prompto.bash`, `src/shell/scripts/prompto.fish`, `src/shell/scripts/prompto.ps1`, `src/shell/scripts/prompto.zsh`, `src/shell/daemon_scripts_test.go`.
- **Spec link:** PLAN Phase E2.

### E3. Manual cross-shell smoke

- **Acceptance:** Vim-mode toggle latency feels identical to `main` across all four shells. Findings recorded in the PR description, signed off by the human.
- **Verify:** Manual. Reviewer triggers `ESC`/`i` repeatedly under each shell with daemon running; latency subjectively matches baseline.
- **Files:** None (PR description only).
- **Spec link:** PLAN Phase E3.

---

## Phase D — Adjacent Subsystems (GATED — placeholders only)

These tasks are NOT scheduled. They become real after the Phase C gate decision. Listing as one-liners for visibility.

- D1. `src/prompt/` daemon-facing edges — engine, layout, streaming.
- D2. `src/config/` daemon-facing edges — layout YAML parsing, watcher contract.
- D3. `src/segments/` daemon-facing edges — segment interface the Registry depends on.
- D4. `src/template/`, `src/runtime/`, `src/cache/` daemon-facing edges.

Each will get a full task breakdown (audit → characterize → refactor → docs) if/when the gate opens.

---

## Phase F — Docs Landing & Wrap

### F1. Finalize `src/daemon/ARCHITECTURE.md`

- **Acceptance:** ARCHITECTURE.md cross-references final code (file names, function names) after all C+E refactors. Diagrams reflect the executed shape. Every C/E PR that drifted from the A2 draft is reconciled.
- **Verify:** Reviewer can read ARCHITECTURE.md cover-to-cover and predict the file layout in `src/daemon/`.
- **Files:** `src/daemon/ARCHITECTURE.md`.
- **Spec link:** SPEC Success Criteria #5.

### F2. Finalize `docs/maintainers/daemon-shells.md`

- **Acceptance:** Each shell section's body is filled in with the post-E2 script contents and pointers to relevant `src/shell/*.go` code.
- **Verify:** Reviewer can debug a shell-integration bug using only this file + the source.
- **Files:** `docs/maintainers/daemon-shells.md`.
- **Spec link:** SPEC Success Criteria #5.

### F3. Banner historical plans

- **Acceptance:** Both `.claude/docs/daemon-vim-mode-plan.md` and `.claude/docs/shell-vim-mode-plan.md` carry a top-of-file banner: "**HISTORICAL** — superseded by `src/daemon/ARCHITECTURE.md`. Kept for provenance." Contents otherwise unchanged.
- **Verify:** Banner text is present and consistent.
- **Files:** `.claude/docs/daemon-vim-mode-plan.md`, `.claude/docs/shell-vim-mode-plan.md`.
- **Spec link:** SPEC Project Structure → Documentation targets.

### F4. Success-criteria audit

- **Acceptance:** Walk SPEC.md's "Success Criteria" section item by item. PR description has a checklist linking each criterion to the commit(s) that close it. Any criterion that cannot be checked is surfaced as a deferred follow-up issue.
- **Verify:** Checklist is fully ✅ or each ❌ has a linked issue.
- **Files:** None (PR description only).
- **Spec link:** SPEC Success Criteria (all).

---

## Open Questions Remaining

Carried from PLAN.md; need answers before the relevant phase starts (not blocking Phase 4 approval):

1. **PLAN Q1** (branch strategy): default = many small PRs against `main`. Confirm.
2. **PLAN Q3** (diagrams): ASCII vs mermaid — answer needed before A2.
3. **PLAN Q4** (`-race` in CI): add during B3 or later? Answer needed before B3.
4. **C2a decision** (Coordinator absorb vs keep): answer needed before C2b.
5. **C6b decision** (lock collapse vs keep): answer falls out of the C6b audit itself.
