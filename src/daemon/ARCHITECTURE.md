---
title: Daemon Architecture
description: Target architecture for the prompto daemon — lifecycle, render
  orchestration, the Hard/Soft cancel model, and per-session update streaming.
---

> **Status:** Target architecture. Phase A2 of the daemon cleanup. Code is
> refactoring toward this shape across the C-series PRs; deviations between
> this doc and the current code are tracked in
> `.claude/notes/daemon-current-state.md`.
> Each C-series PR closes one of those deltas and updates this doc in the
> same commit. Subsumes `.claude/docs/daemon-vim-mode-plan.md` and
> `.claude/docs/shell-vim-mode-plan.md` (both retained as historical).

## What the daemon does

A long-lived background process that renders prompts on behalf of one or more
shell sessions. It is started lazily by the `prompto` CLI, talks to clients
over gRPC on a Unix-domain socket (named pipe on Windows), and stays alive
between commands so that:

- The prompt engine and its config are loaded once, not per keystroke.
- Heavy segments (git, k8s, …) reuse in-flight computation across "repaint"
  requests (vim-mode toggles).
- A per-session cache keeps already-resolved segment text addressable.

## Layered overview

```
+-------------------------------------------------------------+
|  Shell scripts (src/shell/scripts/prompto.{bash,fish,ps1,zsh}) |
|    Detects vim-mode change → calls `prompto render --repaint` |
+----------------------------+--------------------------------+
                             |
                             v
+-------------------------------------------------------------+
|  Client (src/daemon/client.go + client_dial.go + client_rpc.go) |
|    Connect-or-start daemon, fire RPC, parse PromptResponse    |
+----------------------------+--------------------------------+
                             | gRPC over Unix socket / named pipe
                             v
+-------------------------------------------------------------+
|  IPC (src/daemon/ipc/)  —  protocol, sockets, FlagsToProto    |
+----------------------------+--------------------------------+
                             |
                             v
+-------------------------------------------------------------+
|  Server (src/daemon/server.go + server_render.go +            |
|          server_cache.go + server_session.go)                 |
|    gRPC method handlers; per-session toggle store;            |
|    config-reload worker (drives ReloadGate)                   |
+----------------------------+--------------------------------+
                             |
                             v
+-------------------------------------------------------------+
|  Daemon (src/daemon/daemon.go)   ← the only orchestration type |
|    Lifecycle: lock-file, idle timer, on-stop callback         |
|    Owns: EngineRegistry, ReloadGate, DeviceCache,             |
|          ConfigWatcher, BinaryWatcher, ProcessTracker         |
|    API: StartRender, NextUpdate, CompleteSession, Reload,     |
|         Stop, Snapshot                                        |
+----------------------------+--------------------------------+
                             |
                             v
+-------------------------------------------------------------+
|  RenderPipeline (src/daemon/render_pipeline.go)               |
|    Executes ONE render. Returns PromptBundle + ActiveRender.  |
|    Wires engine→hub via BindSegmentUpdates.                   |
|    Honors CancelKind on entry.                                |
+----------------------------+--------------------------------+
                             |
                             v
+----------------------------+----------------+----------------+
|  EngineRegistry            |  SessionUpdateHub               |
|  (registry.go + cancel.go) |  (update_hub.go)                |
|    per-session prompt.Engine    per-session pub/sub of       |
|    + RegistryContext (the      sequence-numbered segment     |
|    Hard/Soft-aware ctx)        updates                        |
+----------------------------+----------------+----------------+
                             |                |
                             v                v
+-------------------------------------------------------------+
|  DeviceCache (cache.go)  +  Session text cache (prompt pkg)   |
|    TTL-based segment-text cache the engine writes to on       |
|    completion; render reads on repaint (the "Fast Path").     |
+-------------------------------------------------------------+
```

**Five files were collapsed to get here.** See the per-file map below for
what each layer owns now. `coordinator.go`, `request_manager.go`,
`runtime.go`, `prompt_cache_bridge.go`, and `service.go` no longer exist as
separate concepts — their responsibilities live on `Daemon` (lifecycle +
orchestration) and `RenderPipeline` (per-render execution).

## The cancel model — first-class `CancelKind`

The single most important concept in this daemon is **which kind of cancel**
is happening when a new request arrives for a session that already has one
in flight. Today, a `bool repaint` parameter flows through six layers; in
this architecture it is a first-class enum at the top of the stack:

```go
// CancelKind classifies why an in-flight render is being interrupted.
// It is the sole signal RenderPipeline and EngineRegistry use to decide
// whether to abort or preserve in-flight segment computations.
type CancelKind int

const (
    // CancelHard means "the on-disk state may have changed."
    // The user pressed Enter, a new command ran, the cwd changed.
    // In-flight computations must abort and MUST NOT write to the cache.
    CancelHard CancelKind = iota

    // CancelSoft means "the on-disk state is unchanged, only the view did."
    // The user toggled vim mode, hit ESC/i, repainted the line.
    // In-flight computations stay alive and their results are reused.
    CancelSoft
)
```

`CancelKind` is derived **once**, at the boundary between `Server` and
`Daemon`, from `PromptRequest.repaint`:

```go
kind := CancelHard
if request.GetRepaint() {
    kind = CancelSoft
}
daemon.StartRender(RenderRequest{..., Cancel: kind})
```

From there, no boolean `repaint` parameter exists in the daemon-internal
API. `RegistryContext` (defined in `cancel.go`) wraps a `context.Context`
with the `CancelKind` that started it:

```go
type RegistryContext struct {
    context.Context
    Kind CancelKind
}
```

Cancellation rules — enforced in `EngineRegistry`:

| New request kind | Action on prior in-flight render |
|---|---|
| `CancelHard` | Cancel `RegistryContext` of prior render. Hub writes from prior render are dropped (`ctx.Err() != nil` gate). |
| `CancelSoft` | Close the prior RPC stream only. **Preserve** `RegistryContext` — in-flight compute keeps running; new request reattaches via `Registry.GetActiveRender`. |

Cache-write safety (mirrored in `cache.go`):

```go
// Any code path that writes to DeviceCache MUST go through this gate.
func (c *DeviceCache) SetIfActive(ctx context.Context, key string, val SegmentRenderValue, ttl time.Duration) {
    if ctx.Err() != nil {
        return // aborted; never pollute the cache
    }
    c.Set(key, val, ttl)
}
```

## The three vim-mode scenarios

The cancel model is best understood by walking the three canonical
scenarios. Each is also a test in `src/daemon/scenarios_test.go` (Phase B2).

### Scenario 1 — Soft cancel during vim toggle

User has a heavy `git status` segment in flight. User presses `ESC` (vim
command mode). Shell calls `prompto render --repaint`.

```
t=0    Request A arrives, kind=CancelHard.
       Daemon.StartRender → RenderPipeline.Start →
         Registry.SetActiveRender(sessionID, ctx, cancel)
       Engine starts segments; git is slow (still running).

t=50ms Request B arrives, kind=CancelSoft.
       Daemon.StartRender →
         Registry.GetActiveRender(sessionID) → (ctx, renderID, ok=true)
       Pipeline reattaches:
         - Closes A's RPC stream (client A exits).
         - Reuses A's RegistryContext (NOT cancelled).
         - Returns initial bundle immediately using cached segment text.

t=80ms git completes in A's still-alive goroutine.
       Hub.Publish("git", renderID) → flows to B's NextUpdate stream.
       Cache write succeeds (ctx.Err() == nil).
```

Net effect: the heavy work runs **once**. User sees an instant repaint.

### Scenario 2 — Hard cancel on new command

User has a heavy `git status` segment in flight (predicting "clean"). User
runs `touch file && git add file`. Shell calls `prompto render` (no
`--repaint`). Repo state on disk has changed; the in-flight `git` result
would be stale.

```
t=0    Request A arrives, kind=CancelHard. git starts, predicts "clean."

t=50ms Request B arrives, kind=CancelHard.
       Daemon.StartRender →
         Registry.CancelActiveRender(sessionID)  ← CancelHard
         => cancel(A.ctx) is called.
       Pipeline starts fresh:
         - Registry.SetActiveRender(...) with new ctx.
         - Engine starts new segments. git restarts, sees "dirty."

t=51ms A's git goroutine wakes, finds ctx.Err() != nil.
       Skips cache write. Hub publish is also gated by ctx.Err().

t=120ms B's git completes ("dirty"). Cache write succeeds.
```

Net effect: stale "clean" result **never** reaches the cache. Safety
preserved.

### Scenario 3 — Rapid-fire toggles

User spams `ESC i ESC i ESC` faster than git can complete.

```
t=0     Request A arrives, kind=CancelHard. git starts.
t=20ms  Request B, kind=CancelSoft. Reattach to A.
t=40ms  Request C, kind=CancelSoft. Reattach to A.
t=60ms  Request D, kind=CancelSoft. Reattach to A.
...
t=100ms git completes (still running in A's context).
        Hub.Publish reaches D's stream (the current subscriber).
```

Net effect: git runs **once** regardless of toggle count.

## Per-file responsibilities (target layout)

Files marked `[NEW]` are introduced during the C-series refactor. Files
marked `[DELETED]` exist today but do not exist post-refactor.

### Wire / protocol — `src/daemon/ipc/`

| File | Owns |
|---|---|
| `daemon.proto` | gRPC service + 8 messages. Every RPC has a doc comment naming its cancel semantics. |
| `daemon.pb.go`, `daemon_grpc.pb.go` | Generated. Never hand-edited. `go generate ./...` must produce no diff in CI. |
| `protocol.go` | `FlagsToProto` / `ProtoToFlags` + `ProtocolVersion` constant. |
| `socket.go` + `socket_unix.go` + `socket_windows.go` | Cross-platform socket-path resolution, listen, dial, cleanup. |

### Server / CLI — `src/daemon/server*.go`, `src/cli/daemon*.go`

| File | Owns |
|---|---|
| `server.go` | gRPC wiring, lifecycle (Start/Stop/Done), helpers. |
| `server_render.go` [NEW] | `RenderPrompt` handler. Derives `CancelKind` from `request.Repaint`; calls `Daemon.StartRender`; streams responses. |
| `server_cache.go` [NEW] | `CacheClear`, `CacheSetTTL`, `CacheGetTTL`. |
| `server_session.go` [NEW] | `ToggleSegment`, `SetLogging`, per-session toggle store, config-reload worker. |
| `cli/daemon.go` | `prompto daemon` subcommands. |
| `cli/daemon_unix.go`, `cli/daemon_windows.go` | Detached-process spawn per OS. |

### Orchestration — `src/daemon/daemon.go`

| File | Owns |
|---|---|
| `daemon.go` | The `Daemon` type. Holds: `EngineRegistry`, `ReloadGate`, `DeviceCache`, `ConfigWatcher`, `BinaryWatcher`, `ProcessTracker`, `LockFile`. Public API: `New`, `StartRender`, `NextUpdate`, `CompleteSession`, `Reload`, `Stop`, `Snapshot`, `SessionCount`, `SessionHub`. |
| `service.go` [DELETED] | Merged into `daemon.go`. No external caller ever held a `*Service`; the type added no value over `*Daemon`. |
| `coordinator.go` [DELETED] | Merged into `render_pipeline.go`. `RenderHandle`'s only consumer was `RequestManager`. |
| `request_manager.go` [DELETED] | Merged into `daemon.go`. `ReloadGate` integration moves directly onto `Daemon.StartRender`. |
| `runtime.go` [DELETED] | Merged into `render_pipeline.go`. `SessionRenderHandle` becomes the existing `ActiveRender` type. |

### Per-render execution — `src/daemon/render_pipeline.go`

| File | Owns |
|---|---|
| `render_pipeline.go` | `RenderPipeline.Start(sessionID, flags, kind CancelKind) → (PromptBundle, *ActiveRender)`. Resolves engine via Registry, applies flags (full vs repaint-only), wires segment-update publish via `BindSegmentUpdates(sessionID, engine, sessionStore)` — no more inline `handle.Hub().Publish()` calls. `ActiveRender.Next(ctx, after)` streams via `StreamRelay`. |

### Registry + cancellation — `src/daemon/registry.go` + `cancel.go`

| File | Owns |
|---|---|
| `cancel.go` [NEW] | `CancelKind` enum + `RegistryContext` type + `DeviceCache.SetIfActive` helper signature documentation. The cancel model in code form. |
| `registry.go` | `EngineRegistry`. Per-session `prompt.Engine` cache + active-render slot (`RegistryContext` + cancel func + renderID). API mirrors the cancel-kind cases: `StartHard(sessionID, flags)`, `StartSoftOrReuse(sessionID, flags)`, `Complete(sessionID, renderID)`, `RemoveSession`, `Reset`. The old `bool repaint` API is gone. |

### Update streaming — `src/daemon/update_*.go` + `stream_relay.go` + `session_store.go`

| File | Owns |
|---|---|
| `update_hub.go` | `SessionUpdateHub` — per-session pub/sub of sequence-numbered snapshots. No changes from today. |
| `update_binding.go` | `BindSegmentUpdates(sessionID, engine, store)` / `ClearSegmentUpdates(engine)`. **Now actually used** by `RenderPipeline.Start`. The inline publishes in render_pipeline.go are removed. |
| `stream_relay.go` | `StreamRelay.Next(ctx, after, renderID)` — hub→ctx replay relay for `NextUpdate`. No changes. |
| `session_store.go` | `PromptSessionStore` — owns hubs per session, delegates engine removal to Registry. No changes. |

### Cache — `src/daemon/cache.go`

| File | Owns |
|---|---|
| `cache.go` | `DeviceCache` directly implements `prompt.DeviceCache` (no bridge). Adds `SetIfActive(ctx, key, val, ttl)` — the only allowed write path for in-flight renders. |
| `prompt_cache_bridge.go` [DELETED] | Was pure indirection. DeviceCache implements the prompt interface natively. |

### Lifecycle + watchers — `src/daemon/{config_watcher,binary_watcher,reload_gate,lock}.go`

| File | Owns |
|---|---|
| `config_watcher.go` | fsnotify-based watcher over the config + dependent files. Calls `onChange(path)` when any change settles past the debounce window. |
| `binary_watcher.go` | fsnotify on the daemon binary's resolved path. Triggers shutdown on change so the next client start picks up a new binary. |
| `reload_gate.go` | `ReloadGate` — counter + CV for "active requests vs reload pending." `BeginReload` waits for in-flight requests to drain; `StartRequest` blocks while reload is in progress. Owned by `Daemon`. |
| `lock.go` | `LockFile` — cross-platform PID-file. Build-tagged platform helpers live in the same file via `//go:build` blocks; the `lock_unix.go` / `lock_windows.go` split is removed unless code volume dictates otherwise (decision deferred to C6b). |

### Process tracking (idle-shutdown) — `src/daemon/`

Kept in the `daemon` package (the originally-planned `idlestop/` subpackage
was skipped after weighing import overhead vs organisational win — the
in-place rename gets ~90% of the clarity).

| File | Owns |
|---|---|
| `process_tracker.go` | `ProcessTracker` (was `SessionManager`). Register/Unregister/Count + `watchProcess` goroutine per PID. Triggers the idle-shutdown callback when no tracked PIDs remain. |
| `process_wait_{linux,macos,freebsd,windows,other}.go` | `waitForProcessExit(ctx, pid)` per OS. |
| `process_wait_freebsd_{32,64}.go` | 32/64-bit kevent `setIdent`. |

### Client — `src/daemon/client*.go`

| File | Owns |
|---|---|
| `client.go` | Public surface: `Client` type, `NewClient`, `ConnectOrStart`, `Close`. Response parsing helpers (`PromptResult`, `ExtractPrompts`). |
| `client_dial.go` [NEW] | Dial + retry logic (was inlined). |
| `client_rpc.go` [NEW] | One thin function per RPC: `RenderPrompt`, `RenderPromptSync`, `ToggleSegment`, `Cache*`, `SetLogging`. |

### Environment — `src/daemon/environment.go`

| File | Owns |
|---|---|
| `environment.go` | `Environment` — wraps per-request env + flags. `UpdateForRepaint(flags, env)` for in-place reuse during Soft cancel. |

## Concurrency model

- **One goroutine per active render**, owned by the engine; lives for the
  duration of the slowest segment.
- **One goroutine per RPC stream**, owned by the gRPC handler in
  `server_render.go`. Closes its context when the client disconnects.
- **One goroutine per tracked PID** in `ProcessTracker`. Exits when the PID exits.
- **One goroutine per watcher** (config, binary). Idle until fsnotify fires.
- **One goroutine for config-reload worker** in `server_session.go`. Reads
  from a queue, calls `Daemon.Reload`.

Shared state is documented per-field with `// guarded by mu` comments
(see `code-style` in `SPEC.md`). All cancellation flows through
`RegistryContext`; no goroutine spawns a background context that isn't
tracked by Registry.

## What this architecture deliberately does **not** do

- No batched rendering across sessions. One session, one render.
- No persistent on-disk cache. `DeviceCache` is in-memory only; `prompt`
  package owns any disk-backed caching.
- No backpressure beyond cancel. If a client cannot keep up with hub
  updates, the relay drops to "replay latest" semantics rather than buffer.
- No multi-daemon coordination. Lock file ensures one daemon per host;
  second start fails fast.

## Where to look first if you're new

1. **`daemon.proto`** — the wire contract. Everything else serves this.
2. **`cancel.go`** — the cancel model in 20 lines.
3. This file's "three vim-mode scenarios" section.
4. **`render_pipeline.go`** — what actually runs when a render starts.
5. **`registry.go`** — where state lives between renders for the same
   session.
