---
title: Runtime Architecture
description: Quick map of the prompto runtime architecture and where to start reading code.
---

## Purpose
This file is a fast entry point for humans and agents.
It explains where core runtime behavior lives after the move to layout-only config.

## High-Level Flow
1. Shell calls `prompto render ...`.
2. CLI talks to daemon (gRPC) for prompt rendering.
3. Daemon creates or reuses a per-session prompt engine.
4. Engine renders from layout config (`prompt`, `rprompt`, `secondary`, `transient`, `rtransient`).
5. Initial result is returned quickly, then async segment updates stream until completion.

## Directory Guide
- `src/config`: Layout YAML parsing, validation, separator normalization, runtime metadata.
- `src/prompt`: Prompt engine, layout block construction, async streaming, repaint behavior.
- `src/daemon`: Session lifecycle, render orchestration, update streaming, config/binary watchers.
- `src/cli`: Command entrypoints and daemon client plumbing.

## First Files To Read
- `src/config/layout.go`
- `src/config/load.go`
- `src/prompt/engine.go`
- `src/prompt/layout.go`
- `src/prompt/streaming.go`
- `src/daemon/server.go`
- `src/daemon/service.go`
- `src/daemon/render_pipeline.go`

## Design Constraints
- Layout YAML is the only runtime config path.
- Top-level keys are strict (`secondary`, `transient`, `rtransient`, `vim-mode`).
- Legacy top-level aliases are rejected with explicit parse errors.
- Render requests are session-scoped and support repaint semantics.
