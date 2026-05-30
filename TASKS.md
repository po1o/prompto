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

### C1a. Introduce `CancelKind` and `RegistryContext` types (additive) — DONE 2026-05-29

- **Acceptance:** New types defined in `src/daemon/cancel.go` (not `registry.go` — follows the `cancel.go` home defined in `ARCHITECTURE.md`): `CancelKind` (enum: `CancelHard`, `CancelSoft`), `RegistryContext` (embeds context.Context + Kind). Also `CancelKindForRepaint(bool)` (the single bool→kind boundary), `CancelKind.Repaint()`, `CancelKind.String()`, `WithCancelKind(ctx, kind)`. All exported so the additive-only commit stays `unused`-lint-clean; referenced by tests. No existing callers migrated.
- **Verify:** Universal gates + `go vet ./daemon/...`. Scenarios tests from B2 still green. ✅ All passed (build, vet, lint 0 issues, fieldalignment, modernize, `-race`).
- **Files:** `src/daemon/cancel.go` [NEW], `src/daemon/cancel_test.go` [NEW].
- **Spec link:** SPEC Code Style example block; PLAN Phase C1.

### C1b / C1c / C1d — FOLDED INTO C2 (2026-05-29)

**Resequenced.** These tasks were written against a hypothetical registry API
(`Subscribe`/`Cancel`/`future`) that does not exist. In the real code the
`EngineRegistry` is cancel-kind-agnostic; the Hard/Soft decision lives in
`coordinator.go`, and the `bool repaint` flows through the 5-layer tower
(`Service → SessionRenderRuntime → RequestManager → RenderCoordinator →
applyRenderFlags`) that C2 deletes. Threading `CancelKind` through doomed
layers is throwaway work. Decision (human-approved): introduce `CancelKind`
as the cancel signal **in the collapsed code**, as part of C2. C1d's original
targets (`update_hub`/`binding`/`relay`) carry no repaint flag and need no
migration — `update_binding.go` is dead code, deleted in C2-iii anyway.

### C2 — Collapse orchestration tower + wire CancelKind

Replaces old C2a/C2b. The post-refactor boundary is already decided and
documented in `ARCHITECTURE.md` (A2): `Daemon` absorbs `Service` +
`RequestManager`; `RenderPipeline` absorbs `RenderCoordinator` +
`SessionRenderRuntime`; `EngineRegistry` gains kind-aware entry points.
Executed as ordered, individually-revertible sub-steps, each behavior-
preserving and gated by the B2 scenario tests + full suite + `-race`.

#### C2-i. Make `EngineRegistry` kind-aware; absorb `RenderCoordinator` — DONE 2026-05-29

- **Acceptance:** `EngineRegistry` gains a kind-aware entry point that
  encapsulates the hard-cancel-then-start vs soft-reattach branch currently
  in `coordinator.go` (e.g. `StartRender(sessionID, flags, kind CancelKind)`
  returning the render handle). `RenderCoordinator`'s logic moves onto the
  registry; `coordinator.go` is deleted (its `RenderHandle` either moves to
  registry or is replaced by the existing handle type). `RequestManager`
  calls the new registry method with a `CancelKind` instead of `bool repaint`.
- **Verify:** Universal gates + B2 scenarios green. `git grep -n 'repaint bool' src/daemon/registry.go src/daemon/request_manager.go` returns nothing.
- **Files:** `src/daemon/registry.go`, `src/daemon/coordinator.go` [DELETE], `src/daemon/request_manager.go`, plus `_test.go` siblings (`coordinator_test.go` cases move to `registry_test.go`).

#### C2-ii. Collapse `SessionRenderRuntime` + `RequestManager` into `RenderPipeline` — DONE 2026-05-30

- **Acceptance:** `RenderPipeline.Start` takes `CancelKind` (not `bool
  repaint`) and owns what `SessionRenderRuntime` + `RequestManager` did
  (engine resolution, ReloadGate gating, handle creation). `runtime.go` and
  `request_manager.go` deleted; `SessionRenderHandle`/`RequestHandle` unified
  with the existing `ActiveRender` type. `Service` calls the new pipeline.
- **Verify:** Universal gates + B2 scenarios green. Net LoC in `src/daemon/` drops. Three handle types reduced to one.
- **Files:** `src/daemon/render_pipeline.go`, `src/daemon/runtime.go` [DELETE], `src/daemon/request_manager.go` [DELETE], `src/daemon/service.go`, plus `_test.go` siblings.

#### C2-iii + C2-iv. Collapse `Service` into `Daemon`; delete dead `update_binding.go`; derive `CancelKind` at the Server boundary — DONE 2026-05-30

Note: C2-iv (server boundary CancelKind derivation) folded into C2-iii since the changes are tightly coupled. Also a course correction during execution: the original plan was "route publish through BindSegmentUpdates," but investigation showed `engine.SetUpdateCallback` (the path BindSegmentUpdates wires) and `PrimaryStreaming`'s inline callback are *separate* channels — wiring both would cause duplicate publishes. Honored the A1 "dead code" finding and deleted update_binding.go outright; PrimaryStreaming's inline callback in render_pipeline.go is the live channel.



- **Acceptance:** `Service`'s methods move onto `Daemon` (the 1:1 wrapper from
  A1/F3 collapses); `service.go` deleted. The inline `handle.Hub().Publish()`
  calls in `render_pipeline.go` route through `BindSegmentUpdates` (per the
  A1 dead-code decision: "use Bind, delete inline"). `RenderRequest` carries
  `CancelKind`.
- **Verify:** Universal gates + B2 scenarios green. `git grep -n 'NewService\|\*Service' src/daemon/` returns nothing.
- **Files:** `src/daemon/daemon.go`, `src/daemon/service.go` [DELETE], `src/daemon/render_pipeline.go`, `src/daemon/update_binding.go`, plus `_test.go` siblings.

#### C2-iv. — FOLDED INTO C2-iii (DONE 2026-05-30)

`bool repaint` survives only in (a) `cancel.go CancelKindForRepaint` (the conversion fn), (b) `client.go` RPC wrappers (passthrough to the proto bool field), and (c) `render_pipeline.go applyRenderFlags` (an internal vim-mode-only flag-mutation helper, not a cancel decision). All three are acceptable per the spec's "no bool in cancel decisions" intent.

### C3. Rename `SessionManager` → `ProcessTracker` (in place) — DONE 2026-05-30

**Revised from original C3a/C3b** which called for subpackaging into
`daemon/sessionid/` (later `idlestop/`). The core smell was the name —
`SessionManager` tracks PIDs, not sessions, and the overloaded "Session"
prefix collided with PromptSessionStore and SessionRenderRuntime (the
latter is already gone after C2). An in-place rename + file rename gets
~90% of the clarity benefit without the import overhead of a new package.

- **What changed:**
  - `SessionManager` type → `ProcessTracker`; `NewSessionManager` → `NewProcessTracker`; receiver `sm` → `tracker`.
  - `session.go` → `process_tracker.go`; `session_test.go` → `process_tracker_test.go`.
  - `session_{freebsd,freebsd_32,freebsd_64,linux,macos,other,windows}.go` → `process_wait_*.go`.
  - `session_store.go` left untouched (`PromptSessionStore` is a different concept — render-hub storage, not PID tracking).
  - ARCHITECTURE.md updated to drop the `idlestop/` subpackage plan.
- **Verify:** Universal gates green. ✅ build, vet, daemon `-race`, lint, fieldalignment, modernize all clean. CI matrix covers all OS-specific waiter files.
- **Files:** `process_tracker.go`, `process_tracker_test.go`, `process_wait_*.go`, `daemon.go` (callers), `service_test.go` (helper), `daemon_test.go` (helper), `ARCHITECTURE.md`.

### C4 — Clean the Server/Daemon boundary, then split `server.go`

**Revised 2026-05-30.** The original C4 only split `server.go` cosmetically by RPC method group. Per review, that leaves the *real* problem unaddressed: `Server` today owns per-session business state (toggles, primary-stream cancellation tracker, the config-reload worker, the config + binary watchers, a duplicate `deviceCache` field) that has nothing to do with the gRPC wire and should live on `Daemon`. Without moving that state, the principle "Daemon = pure-Go business logic; Server = thin wire adapter" is aspirational rather than true. C4 is resequenced into four ordered sub-steps that fix the boundary first, then do the cosmetic split.

#### C4a. Move per-session toggles to `Daemon` — DONE 2026-05-30

- **Acceptance:** `segmentToggles map[string]map[string]bool` + `toggleMu` + `sessionToggles`/`cloneToggleMap` helpers move from `Server` to `Daemon`. Daemon exposes `ToggleSegment(sessionID string, segments []string)` and `SessionToggles(sessionID string) map[string]bool`. `Server.ToggleSegment` RPC handler becomes a thin proxy. Render path reads toggles via `Daemon`, not `Server`.
- **Verify:** Universal gates + B2 scenarios + full daemon -race green. `git grep -n 'segmentToggles\|toggleMu' daemon/server.go` returns nothing.
- **Files:** `src/daemon/daemon.go`, `src/daemon/server.go`, plus `_test.go` siblings.
- **Spec link:** PLAN Phase C4 (revised).

#### C4b. Move config-reload worker + watchers to `Daemon`; dedupe `DeviceCache` — DONE 2026-05-30

- **Acceptance:** `ConfigWatcher`, `BinaryWatcher`, the `configReloadCh` channel, `configReloadWorker` goroutine, `processPendingConfigReload`, `applyConfigReload`, `reloadIfConfigFileUpdated`, `captureConfigModTime`, `requestConfigReload`, `refreshConfigWatches`, `lastConfigModUnixNano`, `reloadMu`, and the `configPath` field all move from `Server` to `Daemon`. `Server.deviceCache` field deleted (was duplicate; `Daemon.DeviceCache()` is the single source). Cache RPC handlers read via `server.core.DeviceCache()`.
- **Verify:** Universal gates green. `git grep -n 'configReloadCh\|configWatcher\|binaryWatcher\|reloadMu\|lastConfigModUnixNano\|deviceCache' daemon/server.go` returns nothing.
- **Files:** `src/daemon/daemon.go`, `src/daemon/server.go`, plus `_test.go` siblings.
- **Spec link:** PLAN Phase C4 (revised).

#### C4c. Split the now-thin `server.go` by RPC method group — SKIPPED 2026-05-30

After C4a+C4b, `server.go` dropped from 560 to 362 LoC and is now coherent
(handlers + wire wiring + helpers). The cosmetic split would trade one tight
file for four small ones — more files to grok, not less. Original C4 motive
(break the 560-LoC overload) was already addressed by C4a/b. Revisit if
`server.go` grows again.



- **Acceptance:** What remains of `Server` is wire-only: gRPC lifecycle (`grpcServer`, `listener`, `done`, `shutdownOnce`, `lockFile`, `Start`/`Stop`/`Done`), per-RPC handlers, the `primaryStreams` cancellation tracker (genuinely gRPC-specific — it forces wire-side handlers to return when a session's stream is superseded), and proto↔Go conversion at the boundary. Split into `server.go` (wiring/lifecycle), `server_render.go` (`RenderPrompt`), `server_session.go` (`ToggleSegment`, `SetLogging`), `server_cache.go` (`CacheClear`/`SetTTL`/`GetTTL`). No method count change.
- **Verify:** Universal gates green. `wc -l src/daemon/server*.go` shows the new layout. `go doc ./daemon | grep -c '^func.*Server'` matches `main`.
- **Files:** `src/daemon/server.go` + 3 new `server_*.go` files, plus `server_test.go` (split if natural).
- **Spec link:** PLAN Phase C4 (revised).

#### C4d. Align `src/cli/daemon*.go` to the new Server shape — N/A 2026-05-30

Server's external API (`NewServer(configPath)`, `Start`, `Stop`, `Done`)
did not change in C4a/b. No CLI updates required. Verified by `grep` —
`cli/daemon*.go` only touches `NewServer`, `Start`, `Stop`, `Done`.



- **Acceptance:** CLI plumbing files reflect any naming changes from C4a–c. No new flags, no removed flags.
- **Verify:** Universal gates green. `prompto daemon --help` output matches `main`'s verbatim (capture both, diff).
- **Files:** `src/cli/daemon.go`, `src/cli/daemon_unix.go`, `src/cli/daemon_windows.go`.
- **Spec link:** PLAN Phase C4 (revised).

#### Note: `primaryStreams` stays on Server (with caveat)

`primaryStreams` + `replacePrimaryStream` track per-session gRPC stream cancel funcs so that when a new render arrives for a session, the prior gRPC handler returns immediately (rather than waiting for the next `daemon.NextUpdate` poll to notice the render was superseded). This **is** gRPC-handler-specific and correctly belongs on Server. **Open follow-up (post-C4):** A1 finding #5 noted this is a *parallel* cancellation layer on top of Registry's own active-render slot — there may be a way to fold the two into one mechanism. Out of scope for C4; revisit if it ever causes a real bug.

### C5. Simplify `client.go` and add tests — DONE 2026-05-30 (split skipped)

**Revised.** The original C5 was a 3-way file split (client.go +
client_dial.go + client_rpc.go), motivated by the file's 336 LoC. Like
C4c, the file is actually well-organized (types → connection → RPCs →
helpers → result extraction); splitting scatters related code without a
real win. The genuine smells were elsewhere:

- **Simplified `ExtractPrompts`**: 9 near-identical if-blocks → one loop
  over a `{dst, key}` table. Data, not code.
- **Added `client_test.go`**: 5 tests for `ExtractPrompts` (nil, empty,
  all-fields, partial, unknown-keys). Brings ExtractPrompts from 0% → 100%
  coverage. Closes one of B1's biggest gaps.
- The connection / RPC wrappers still need an in-process gRPC harness to
  test meaningfully (heavy lift); they remain at 0% coverage. Noted as a
  follow-up — covering them would require a substantial new test scaffold.
- **Files:** `src/daemon/client.go`, `src/daemon/client_test.go` [NEW].

### C6. Watchers + lock — NO ACTION 2026-05-30

A1 flagged `ReloadGate` vs `ConfigWatcher` as overlapping, but closer
reading shows they have distinct, complementary roles:

- **ReloadGate** is an active-requests counter: `BeginReload` blocks while
  any request is in-flight; `StartRequest` blocks while a reload is pending.
  It says nothing about *when* to reload.
- **ConfigWatcher** is an fsnotify wrapper: it emits "config X changed"
  events. It says nothing about *what to do* when one fires.
- They meet in `Daemon.configReloadWorker` (`daemon_reload.go`), which
  receives ConfigWatcher's events and runs `Daemon.Reload`, which in turn
  uses ReloadGate to serialize against active requests. Clean composition.

The `lock_unix.go` + `lock_windows.go` split is idiomatic Go (the
`_GOOS.go` suffix is the standard implicit build constraint). Per-function
build tags don't exist in Go, so collapsing into one file isn't possible.
The split is load-bearing.

No code action needed. Documenting the analysis here so the A1 finding is
explicitly closed.

### C7. Resolve `prompt_cache_bridge.go` indirection — DONE 2026-05-30

`SegmentRenderValue` is now a type alias for `prompt.DeviceCacheEntry`, so
`*DeviceCache` satisfies `prompt.DeviceCache` directly. The bridge file is
deleted; the `daemon.go` constructor passes `deviceCache` straight into
`NewRenderPipeline`. Color fields (`Foreground`, `Background`) were already
compatible (`color.Ansi` is `type Ansi string`) — string-literal tests
continued to compile unchanged.

- **Files:** `src/daemon/cache.go`, `src/daemon/daemon.go`, `src/daemon/prompt_cache_bridge.go` [DELETED].

### C8. Add proto doc comments + environment audit — DONE 2026-05-30

- `daemon.proto`: expanded the `RenderPrompt` RPC comment to spell out the
  Hard/Soft cancel semantics tied to the `repaint` field, with a pointer to
  ARCHITECTURE.md.
- `environment.go`: audited (61 LoC, well-structured); added a one-line
  cancel-model pointer on `UpdateForRepaint`.
- `go generate ./daemon/ipc/` produced no changes to generated stubs (doc
  comments don't affect generated Go code).
- **Files:** `src/daemon/ipc/daemon.proto`, `src/daemon/environment.go`.

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
