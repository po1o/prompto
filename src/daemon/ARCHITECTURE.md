---
title: Daemon Architecture
description: Session-aware daemon lifecycle, render orchestration, and update streaming model.
---

## Responsibility
`src/daemon` runs the long-lived rendering service and streams prompt updates to clients.

## Entry Point
- gRPC server is in `src/daemon/server.go`.
- Protocol definitions are in `src/daemon/ipc/daemon.proto`.

## Render Pipeline
1. `RenderPrompt` receives request with flags/session info.
2. `Server` resolves session id and forwards request to daemon core.
3. `Service.StartRender` starts or replaces active render stream for the session.
4. Initial bundle is sent immediately.
5. `NextUpdate` streams segment completion updates until render-complete marker.
6. Final `complete` response is sent.

## Core Components
- `Daemon` (`src/daemon/daemon.go`): lifecycle, idle shutdown policy, session tracking.
- `Service` (`src/daemon/service.go`): manages active renders per session.
- `RenderPipeline` (`src/daemon/render_pipeline.go`): executes full render vs repaint strategy.
- `SessionRenderRuntime` and registry files: per-session engines/hubs and coordination.

## Repaint Semantics
- Repaint requests do not restart all segment computations.
- Existing stream context is reused.
- Vim-driven updates are refreshed while other pending/completed segments keep current state.

## Reload and Watchers
- `ConfigWatcher`: triggers coalesced reload signals.
- `BinaryWatcher`: stops daemon when binary changes so next client start uses new binary.
- Reload gate/runtime synchronization prevents config reload from racing active render state.

## Cache and Toggles
- Device cache and session toggles are daemon-owned.
- Cache RPC endpoints expose clear/get/set-ttl operations.
- Segment toggles are session scoped.
