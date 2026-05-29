# Plan: Daemon Mode Cleanup & Documentation

Status: DRAFT — pending human approval before Phase 3 (Tasks).
References: [SPEC.md](./SPEC.md).
Last updated: 2026-05-23.

## Reading Note

The spec's in-scope list now includes `src/prompt/`, `src/config/`,
`src/segments/`, `src/template/`, `src/cache/`, `src/runtime/` in addition to
the daemon proper. This plan treats those as a **separate program phase (D)**
behind a human go/no-go gate, because cleaning them up touches the whole
runtime, not just the daemon. If you want to descope, the natural cut line is
"finish through Phase C + E + F, defer Phase D."

## Major Components (what we're cleaning up)

Grouped by layer, with current size and current sin:

| # | Component | Files | LoC | Today's smell |
|---|---|---|---|---|
| 1 | **IPC wire** | `src/daemon/ipc/*` | gen + 1 proto | Proto under-commented; generated stubs OK. |
| 2 | **Server + CLI** | `src/daemon/server.go`, `src/cli/daemon*.go` | ~870 | Server.go is 560 LoC — too many responsibilities in one file. |
| 3 | **Service + RenderPipeline + Coordinator** | `service.go`, `render_pipeline.go`, `coordinator.go` | ~617 | Overlapping orchestration roles; unclear which owns what. |
| 4 | **Session runtime + store** | `runtime.go`, `session*.go`, `session_store.go` | ~530 | Cross-platform `session_*.go` zoo + a Store + a Session = three concepts blurred. |
| 5 | **Registry + cancellation** | `registry.go`, `request_manager.go`, `update_hub.go`, `update_binding.go`, `stream_relay.go` | ~440 | The Hard/Soft cancel model lives here but isn't expressed in types. |
| 6 | **Cache** | `cache.go`, `prompt_cache_bridge.go`, plus `src/cache/` | ~176 + adj. | Bridge naming hints at an impedance mismatch worth resolving. |
| 7 | **Lifecycle + watchers** | `daemon.go`, `config_watcher.go`, `binary_watcher.go`, `reload_gate.go`, `lock*.go` | ~777 | ReloadGate vs ConfigWatcher overlap; lock_* trio split confusingly. |
| 8 | **Client** | `client.go` | 336 | One file does dial + retry + RPC wrappers + parsing — split candidates obvious. |
| 9 | **Environment** | `environment.go` | 61 | Small; mostly fine. Audit only. |
| 10 | **Shell integration (Go)** | `src/shell/{init,bash,fish,pwsh,zsh}.go` | ~? | Per-shell helpers; check for duplication. |
| 11 | **Shell integration (scripts)** | `src/shell/scripts/prompto.*` | embedded | Where `--repaint`, vim hooks live. Hand-written, untyped, brittle. |
| 12 | **Adjacent subsystems** | `src/prompt/`, `src/config/`, `src/segments/`, `src/template/`, `src/cache/`, `src/runtime/` | large | Out-of-spec until Phase D gate. |

Documentation deliverables:

| Doc | Path | Status |
|---|---|---|
| Canonical daemon arch | `src/daemon/ARCHITECTURE.md` | Rewrite (currently 40 lines, summary-only) |
| Per-shell integration guide | `docs/maintainers/daemon-shells.md` | New file |
| Historical plans | `.claude/docs/{daemon,shell}-vim-mode-plan.md` | Add HISTORICAL banner |

## Phases & Dependency Order

```
A. Audit + target shape    ──┐
                             ├──→  B. Test baseline  ──→  C. Daemon core (sequential)
                             │                                  │
                             │                                  ├──→  E. Shell integration
                             │                                  │
                             │                                  ├──→  [GATE]  D. Adjacent subsystems
                             │                                  │
                             │                                  └──→  F. Docs landing
                             │
                             └──→  (informs all later phases)
```

Sequential edges are dependencies; sibling branches can run in parallel.

### Phase A — Audit + Target Shape (foundation; sequential)

**Goal:** lock down what we have and what we want before changing code.

- **A1. Read pass.** Walk every in-scope file (components 1–11). Produce a one-line "what this file does today" map in conversation context (no committed artifact unless useful).
- **A2. Draft target `src/daemon/ARCHITECTURE.md`.** Define the end-state shape: layer responsibilities, the Hard/Soft cancel model as a first-class type, the three vim-mode scenarios as diagrams. Refactors in Phase C must move the code toward this doc.
- **A3. Draft `docs/maintainers/daemon-shells.md` skeleton.** One section per shell with the contract (mode-detection mechanism, redraw trigger, `--repaint` semantics, ble.sh dependency). Body fills in during Phase E.

**Verification:** human reviews target ARCHITECTURE.md and shell-guide skeleton. Go/no-go before Phase B.

### Phase B — Test Baseline (sequential after A)

**Goal:** before any refactor, pin current behavior with characterization tests so refactors are demonstrably safe.

- **B1. Coverage audit.** For every component 1–11, list what's tested and what isn't. Use `go test -cover` per package.
- **B2. Characterization tests for the three vim-mode scenarios.** End-to-end in `server_test.go` or new `daemon/scenarios_test.go`: Soft cancel, Hard cancel, Rapid toggles. These tests pass on `main` *today* and must keep passing through every later commit.
- **B3. Race detector baseline.** Run `go test -race ./daemon/...` on `main`. If anything fails, file separately — do not fix as part of this effort. We need a known-clean baseline before refactoring concurrency.

**Verification:** characterization tests merged to `main` (or to the long-running cleanup branch) and green under `-race`. Baseline locked.

### Phase C — Daemon Core Refactor (sequential between sub-phases; each its own PR)

Each PR is independently revertible. Each ends with full CI + `-race` green and no behavior change.

- **C1. Registry + cancellation (component 5).** Make Hard/Soft cancel a first-class type (`CancelKind` enum, explicit `RegistryContext`). Split `request_manager.go` if it carries unrelated concerns. *Smallest, most critical, most tested already — start here to validate the workflow.*
- **C2. Service + RenderPipeline + Coordinator (component 3).** Once Registry's contract is clean, collapse overlapping orchestration. Aim for: Service = "what render is active per session", RenderPipeline = "how a single render unfolds", Coordinator = (probably) absorbed into one of the two.
- **C3. Session runtime + store (component 4).** Disentangle Session-the-runtime from Session-the-store from Session-the-OS-detection (`session_{linux,macos,…}.go`). Likely outcome: `runtime.go` stays, `sessionid/` subpackage for OS detection, `session_store.go` keeps current shape.
- **C4. Server + CLI surface (component 2).** Now that callees are clean, split `server.go` by RPC method group. `cli/daemon*.go` aligned to the new server shape.
- **C5. Client (component 8).** Split dial vs retry vs RPC wrappers.
- **C6. Lifecycle + watchers + lock (component 7).** Resolve ReloadGate vs ConfigWatcher overlap. Collapse the `lock_*.go` trio if possible.
- **C7. Cache + bridge (component 6).** Resolve the bridge — either remove the indirection or name it for what it actually does.
- **C8. Environment + IPC proto comments (components 1, 9).** Cleanup pass. Add doc comments to `daemon.proto`; regen stubs; verify no diff.

**Parallelism:** C5, C6, C7, C8 can run in parallel after C4 lands. C1–C4 are strictly sequential because each defines a contract its successors depend on.

**Verification per PR:** `go test -count=1 ./...` + `golangci-lint run` + `fieldalignment` + `modernize` + `go test -race ./daemon/...` + diff-check of `daemon.proto` and CLI flags. PR description links to the spec section it advances.

### Phase E — Shell Integration (parallel with Phase D after Phase C)

- **E1. `src/shell/{init,bash,fish,pwsh,zsh}.go`.** Audit for duplication; align the four shell helpers to a single shape.
- **E2. Embedded scripts (`prompto.{bash,fish,ps1,zsh}`).** Add header comments explaining the `--repaint` wire-up and the mode-detection mechanism. Snapshot tests in `daemon_scripts_test.go` lock the script contents per shell.
- **E3. Manual smoke.** Cross-shell verification of vim-mode toggle latency. *This is the only step in the whole plan we can't automate;* defer to the human at PR review.

**Verification:** `daemon_scripts_test.go` green; manual smoke recorded in PR.

### Phase D — Adjacent Subsystems (GATE; only if green-lit after Phase C)

This phase is large. Hold a human go/no-go meeting after Phase C lands. If we proceed, each subsystem gets the same treatment as the daemon proper, but with a tighter rule: **only the daemon-facing edges**, not whole-subsystem rewrites.

- **D1. `src/prompt/`** — engine, layout, streaming. The daemon's main collaborator.
- **D2. `src/config/`** — layout YAML parsing. Touched by ConfigWatcher.
- **D3. `src/segments/`** — segment interface boundary the Registry depends on.
- **D4. `src/template/`, `src/runtime/`, `src/cache/`** — utility layers. Smaller scope.

**Verification per sub-phase:** same gate as Phase C PRs.

### Phase F — Docs Landing & Wrap (sequential, last)

- **F1. Finalize `src/daemon/ARCHITECTURE.md`** with code references that survived the refactor. Diagrams in ASCII or mermaid (per repo norms — TBD in Phase A).
- **F2. Finalize `docs/maintainers/daemon-shells.md`** with code references that survived Phase E.
- **F3. Banner `.claude/docs/{daemon,shell}-vim-mode-plan.md`** — top-of-file pointer to the canonical doc; keep contents for provenance.
- **F4. Spec success-criteria audit.** Re-read SPEC.md `Success Criteria` section, confirm each box.

**Verification:** the success-criteria checklist in SPEC.md is fully checked; PR description links each criterion to the commit that closes it.

## Risks & Mitigations

| # | Risk | Mitigation |
|---|---|---|
| R1 | **Scope creep** — the Phase D expansion is the entire runtime. | Hard gate after Phase C; default is "ship C+E+F, defer D." |
| R2 | **Concurrency regressions** during refactor of Registry / cancel. | Phase B locks a `-race` baseline first; every C PR must pass `-race`; characterization tests cover the three canonical scenarios. |
| R3 | **Generated-code drift** if proto comments touched without regen. | `go generate ./...` + git-diff check is a per-PR gate; CI already enforces. |
| R4 | **Shell behavior regressions** undetected by Go tests. | Script-content snapshot tests in `daemon_scripts_test.go` + manual cross-shell smoke at E3. |
| R5 | **Cross-PR API ripples** — interface changes cascade through callers. | Phase A1 audit identifies all cross-cuts before C1 starts; PR descriptions list "callers touched." |
| R6 | **Doc-vs-code drift** during the long refactor. | Doc lives in same commit as the code it describes. ARCHITECTURE.md updated on every C PR that contradicts it. |
| R7 | **"Net abstraction count must not increase"** turns out too strict and blocks a legitimate split. | Treat the rule as default, not absolute; surface exceptions in the PR description with justification. (Resolves Q2 in SPEC's pushback.) |

## Parallelism Summary

- **Strictly sequential:** A1 → A2/A3 → B → C1 → C2 → C3 → C4 → (Phase C done) → [GATE] → F
- **Parallel after C4:** C5, C6, C7, C8 (all daemon-internal cleanup, no cross-dependencies)
- **Parallel after Phase C:** E (shell) and F1/F2 doc-drafting can both run; D requires the gate.
- **Parallel within D (if gated open):** D1, D2, D3, D4 can each run independently but each touches a separate subsystem owner — coordinate accordingly.

## Verification Checkpoints

| After | Checkpoint | Sign-off |
|---|---|---|
| A | Target ARCHITECTURE.md & shell-guide skeleton reviewed | Human go/no-go |
| B | Characterization tests + `-race` baseline merged | CI green on `main` |
| Each C/D/E PR | Full CI + `-race` + reviewer approval + spec-link in description | PR merged |
| C done | Daemon proper cleanup complete | Human go/no-go for D |
| F | SPEC.md success criteria all ✅ | Effort closed |

## Open Questions for Phase 3 (Tasks)

1. **Single long-lived branch vs. PRs against `main`?** Long-running cleanup branches drift; many small PRs against `main` add review overhead. Plan defaults to many small PRs against `main`. Confirm.
2. **Phase D scope decision** — do we want to commit to Phase D now (plan tasks for it) or hold off until after C? Plan defaults to **plan A–C+E+F now, plan D after the gate**.
3. **Diagrams in ARCHITECTURE.md** — ASCII (no tooling, works everywhere) or mermaid (better visuals, repo-norm TBD)? Surface during Phase A2 if there's no existing precedent.
4. **`-race` in CI** — add it during this effort (Phase B3 deliverable) or punt? SPEC listed under "Ask first: CI config" — needs an answer before Phase B finalizes.
5. **Estimated calendar time:** A ≈ 1 day, B ≈ 2 days, C ≈ 2–3 weeks (8 PRs, ~2 days each), E ≈ 3 days, F ≈ 1 day. Total without D: ~4 weeks of focused work. With D: 8–12 weeks. Realistic against your other commitments?
