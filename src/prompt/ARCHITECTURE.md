---
title: Prompt Engine Architecture
description: How the prompt engine builds layout blocks and renders sync, async, and repaint flows.
---

## Responsibility
`src/prompt` renders all prompt text from `Config.Layout` and runtime environment flags.

## Core State
- `Engine` in `src/prompt/engine.go` owns render state, caches, and providers.
- `Engine.LayoutConfig` is the source of layout lines and named segments.

## Layout Construction
`src/prompt/layout.go` turns a `PromptLayout` into runtime blocks:
- `layoutBlock(...)` clones named segments from `LayoutConfig.Segments`.
- Right-aligned segments mirror separators for visual correctness.
- `layoutPrimaryBlocks(...)` builds all primary left/right lines.

## Primary Rendering
- Entry: `Engine.Primary()` in `src/prompt/primary.go`.
- Writes layout-based primary + rprompt output.
- Applies shell integration markers, title, final space, iTerm features, and optional PWD output.

## Async Streaming
- Entry: `PrimaryStreaming(...)` in `src/prompt/streaming.go`.
- Starts segment executions concurrently.
- Returns quickly after timeout using pending placeholders.
- Streams updates as segments complete.

## Repaint Behavior
- Entry: `PrimaryRepaint()` in `src/prompt/streaming.go`.
- Re-evaluates vim segment only.
- Other segments reuse completed/pending state from cache and active stream context.

## Extra Prompts
- `ExtraPrompt(...)` in `src/prompt/extra.go` supports layout-based `secondary` and `transient`.
- `RPrompt()` in `src/prompt/rprompt.go` renders layout `rprompt` lines.
