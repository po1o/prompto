---
title: Config Architecture
description: How layout YAML is parsed, normalized, validated, and applied to runtime config.
---

## Responsibility
`src/config` owns loading and validating the runtime layout config.

## Main Types
- `LayoutConfig` in `src/config/layout.go`: Canonical parsed representation of layout YAML.
- `Config` in `src/config/config.go`: Runtime metadata/features container used by engine and daemon.

## Load Path
1. `config.Parse(...)` in `src/config/load.go` resolves path and reads YAML.
2. YAML is parsed by `ParseLayoutYAML(...)` in `src/config/layout.go`.
3. `LayoutConfig.ApplyMetadata(...)` copies top-level metadata to `Config`.
4. `Config.Layout` points at the parsed `LayoutConfig` for rendering.

## Layout Sections
- `prompt`: left-aligned primary lines.
- `rprompt`: right-aligned primary lines.
- `secondary`: extra prompt lines for secondary prompt.
- `transient`: extra prompt lines for transient prompt.
- `rtransient`: right-aligned transient lines.

## Strict Top-Level Keys
Parser rejects legacy aliases and asks for canonical names:
- Reject `secondary_prompt` -> use `secondary`.
- Reject `transient_prompt` -> use `transient`.
- Reject `transient_rprompt` -> use `rtransient`.
- Reject top-level `vim` settings -> use `vim-mode`.

## Separator Normalization
`normalizePromptLayouts` and segment normalization convert style shortcuts and explicit separator options into
final diamonds/separators used by rendering.

## Important Metadata Consumed
- `palette`, `palettes`, `maps`, `var`, `upgrade`, `cycle`, `iterm_features`
- `vim-mode`, `daemon_timeout`, `daemon_idle_timeout`
- `render_pending_icon`, `render_pending_background`
- `console_title_template`, `pwd`, `terminal_background`, `shell_integration`, `final_space`
