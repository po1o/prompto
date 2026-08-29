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

## Vim Mode Switching

A vim mode change re-renders exactly one segment. `PrimaryRepaint` reuses the
cached text of every other segment (`streaming.go`, the `repaintOnly` branch),
so the render itself is trivial — but asking for it over the wire is not. The
shell would spawn a process, pay Go and gRPC startup, and make a round trip, all
to redraw one segment. That is roughly 100% overhead.

So the daemon precomputes it. When a render completes, `addVimVariants` renders
the prompt under each mode in `vimVariantModes` and puts the results in
`PromptBundle.Extras` under `vim.<mode>.primary` and `vim.<mode>.right`. These
travel in the existing `prompts` map — no protocol change — and the client emits
them as `vim.<mode>.primary:` lines. The shell keeps them in
`_prompto_vim_prompts` and the keymap hook swaps the strings, so a mode change
costs no process and no IPC.

Three things this depends on:

- **Only on a completed render.** `VimVariants` goes through `PrimaryRepaint`,
  which sets `vimRepainted` and so makes late async vim results be dropped
  (`mergeStreamingResultLocked`). A cold start still needs that merge, so
  computing variants mid-render would lose the initial vim state.
- **Two exit paths.** A render with nothing pending completes inside
  `RenderPipeline.Start` and never reaches `Next`, so both attach the variants.
  Missing the first is why they appear to work and then silently do not.
- **Modes not precomputed still work.** `vimVariantModes` holds the two the
  keymap hook toggles between constantly. Visual and replace fall through to
  `--repaint`, which is unchanged. The shell clears the cache on every
  non-repaint render, so variants cannot outlive the prompt they belong to.
