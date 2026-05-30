---
title: Daemon Architecture
description: Target architecture for the prompto daemon — lifecycle, render
  orchestration, the Hard/Soft cancel model, and per-session update streaming.
---


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
|  Server (src/daemon/server.go) — thin gRPC adapter            |
|    All six RPC handlers, lifecycle (Start/Stop/Done),         |
|    proto↔Go conversion, primaryStreams cancel tracker         |
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

See the per-file map below for what each layer owns. The render path is
intentionally flat: `Daemon` owns lifecycle + orchestration; `RenderPipeline`
owns per-render execution; `EngineRegistry` owns per-session cancellation.

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

Cache-write safety: every cache write in the render path is preceded by a
context-error check at the call site (e.g. in `PrimaryStreaming`'s segment
callback). The pattern is straightforward enough that we deliberately
*did not* add a `SetIfActive` helper — adding a helper that wraps two
lines of code obscures rather than helps, and forces callers to thread an
extra context through. If this hand-discipline starts to slip, that's the
time to introduce the helper.

## The three vim-mode scenarios

The cancel model is best understood by walking the three canonical
scenarios. Each is also a test in `src/daemon/scenarios_test.go`.

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

## Per-file responsibilities

### Wire / protocol — `src/daemon/ipc/`

| File | Owns |
|---|---|
| `daemon.proto` | gRPC service + 8 messages. Every RPC has a doc comment naming its cancel semantics. |
| `daemon.pb.go`, `daemon_grpc.pb.go` | Generated. Never hand-edited. `go generate ./...` must produce no diff in CI. |
| `protocol.go` | `FlagsToProto` / `ProtoToFlags` + `ProtocolVersion` constant. |
| `socket.go` + `socket_unix.go` + `socket_windows.go` | Cross-platform socket-path resolution, listen, dial, cleanup. |

### Server / CLI — `src/daemon/server.go`, `src/cli/daemon*.go`

| File | Owns |
|---|---|
| `server.go` | Thin gRPC adapter. Owns: gRPC lifecycle (`grpcServer`, `listener`, `Start`/`Stop`/`Done`, `lockFile`, `shutdownOnce`), all six RPC handlers, proto↔Go conversion at the boundary (`CancelKindForRepaint`), and `primaryStreams` — the per-session gRPC stream-cancel tracker that forces wire handlers to return immediately when a session is superseded. **All business state lives on `Daemon`.** |
| `cli/daemon.go` | `prompto daemon` subcommands. |
| `cli/daemon_unix.go`, `cli/daemon_windows.go` | Detached-process spawn per OS. |

### Orchestration — `src/daemon/daemon.go` + `daemon_reload.go`

| File | Owns |
|---|---|
| `daemon.go` | The `Daemon` type. Holds: `RenderPipeline`, `ProcessTracker`, `DeviceCache`, `ConfigWatcher`, `BinaryWatcher`, `segmentToggles` map, active-`renders` map, lock-file state. Public API: render orchestration (`StartRender`, `NextUpdate`, `CompleteSession`, `Reload`, `Reset`, `Snapshot`, `SessionCount`, `SessionHub`), lifecycle (`Stop`, `StopSilently`, `SetOnStop`), toggles (`SessionToggles`, `ToggleSegment`, `ResetToggles`), cache (`DeviceCache`), config (`ConfigPath`). |
| `daemon_reload.go` | Config-reload pipeline: `startReloadAndWatchers` wires `ConfigWatcher` + `BinaryWatcher` + the reload-worker goroutine; `ProcessPendingConfigReload` / `ReloadIfConfigFileUpdated` are the exported entry points the gRPC handler calls before each render. |

### Per-render execution — `src/daemon/render_pipeline.go`

| File | Owns |
|---|---|
| `render_pipeline.go` | `RenderPipeline.Start(sessionID, flags, kind CancelKind) → (PromptBundle, *ActiveRender)`. Owns the `ReloadGate`, `EngineRegistry`, and `PromptSessionStore` (per-session hubs). Resolves engine via Registry, applies flags (full vs repaint-only), publishes segment updates inline from `PrimaryStreaming`'s callback. `ActiveRender` bundles the render generation (engine, context, render ID), the update hub + relay, and the reload-gate release; `Next(ctx, after)` streams via `StreamRelay` and `Complete` is idempotent. |

### Registry + cancellation — `src/daemon/registry.go` + `cancel.go`

| File | Owns |
|---|---|
| `cancel.go` | `CancelKind` enum (`CancelHard`, `CancelSoft`) + `RegistryContext` wrapper + `CancelKindForRepaint(bool)` (the single bool→kind boundary at the Server↔Daemon edge) + `CancelKind.Repaint()` predicate. |
| `registry.go` | `EngineRegistry`. Per-session `prompt.Engine` cache + active-render slot (context + cancel func + renderID). One kind-aware entry point: `StartRender(sessionID, flags, kind CancelKind) → *RenderHandle` — soft kind reattaches to the live render, hard kind aborts the prior and starts a new one. Plus `RenderHandle` (the registry-owned generation handle: `Complete`, `RenderID`). |

### Update streaming — `src/daemon/update_hub.go` + `stream_relay.go` + `session_store.go`

| File | Owns |
|---|---|
| `update_hub.go` | `SessionUpdateHub` — per-session pub/sub of sequence-numbered snapshots. |
| `stream_relay.go` | `StreamRelay.Next(ctx, after, renderID)` — hub→ctx replay relay for `NextUpdate`. |
| `session_store.go` | `PromptSessionStore` — owns hubs per session, delegates engine removal to Registry. Owned by `RenderPipeline`. |

### Cache — `src/daemon/cache.go`

| File | Owns |
|---|---|
| `cache.go` | `DeviceCache` — in-memory TTL store. `SegmentRenderValue` is a type alias for `prompt.DeviceCacheEntry`, so `*DeviceCache` satisfies `prompt.DeviceCache` directly with no bridge. |

### Lifecycle + watchers — `src/daemon/{config_watcher,binary_watcher,reload_gate,lock}.go`

| File | Owns |
|---|---|
| `config_watcher.go` | fsnotify-based watcher over the root config + extends. Calls `onChange(path)` when an event settles past the debounce window. Owned by `Daemon`. |
| `binary_watcher.go` | fsnotify on the daemon binary's resolved path. Triggers `Daemon.Stop` on change so the next client start picks up the new binary. Owned by `Daemon`. |
| `reload_gate.go` | `ReloadGate` — counter + CV for "active requests vs reload pending." `BeginReload` waits for in-flight requests to drain; `StartRequest` blocks while reload is in progress. Owned by `RenderPipeline`. Composes with `ConfigWatcher` via `Daemon.configReloadWorker` (in `daemon_reload.go`). |
| `lock.go` + `lock_unix.go` + `lock_windows.go` | `LockFile` — cross-platform PID-file. The `_GOOS.go` suffix is Go's standard implicit build constraint; per-function build tags don't exist, so the three-file split is load-bearing. |

### Process tracking (idle-shutdown) — `src/daemon/`

| File | Owns |
|---|---|
| `process_tracker.go` | `ProcessTracker` (was `SessionManager`). Register/Unregister/Count + `watchProcess` goroutine per PID. Triggers the idle-shutdown callback when no tracked PIDs remain. |
| `process_wait_{linux,macos,freebsd,windows,other}.go` | `waitForProcessExit(ctx, pid)` per OS. |
| `process_wait_freebsd_{32,64}.go` | 32/64-bit kevent `setIdent`. |

### Client — `src/daemon/client.go`

| File | Owns |
|---|---|
| `client.go` | `Client` type + `NewClient`/`ConnectOrStart`/`Close` (connection lifecycle), the RPC wrappers (`RenderPrompt`, `RenderPromptSync`, `ToggleSegment`, `Cache*`, `SetLogging`), and response parsing (`PromptResult`, `ExtractPrompts` — table-driven). |

### Environment — `src/daemon/environment.go`

| File | Owns |
|---|---|
| `environment.go` | `Environment` — wraps per-request env + flags. `UpdateForRepaint(flags, env)` for in-place reuse during Soft cancel. |

## Concurrency model

- **One goroutine per active render**, owned by the engine; lives for the
  duration of the slowest segment.
- **One goroutine per RPC stream**, owned by the gRPC handler in
  `server.go`. Closes its context when the client disconnects.
- **One goroutine per tracked PID** in `ProcessTracker`. Exits when the PID exits.
- **One goroutine per watcher** (config, binary). Idle until fsnotify fires.
- **One goroutine for config-reload worker** in `daemon_reload.go`. Reads
  from `Daemon.configReloadCh`, calls `Daemon.applyConfigReload`.

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
