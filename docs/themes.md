---
title: Themes
description: Use the bundled themes as starting points for your own local prompto configuration.
---

## Where Themes Live

Bundled themes are stored in [`themes/`](../themes).
They are plain YAML files, and prompto also compiles them into the binary for `config list` and `config set`.

Examples:

- [`themes/agnoster.minimal.prompto.yaml`](../themes/agnoster.minimal.prompto.yaml)
- [`themes/tokyo.prompto.yaml`](../themes/tokyo.prompto.yaml)
- [`themes/powerlevel10k_modern.prompto.yaml`](../themes/powerlevel10k_modern.prompto.yaml)

## Recommended Workflow

1. Pick a theme that is visually close to what you want.
2. Write it to the default config path.
3. Point shell init at that local file.
4. Edit the copy instead of editing the theme in-place.

List the bundled themes:

```bash
prompto config list
```

Write one to the default config path:

```bash
prompto config set tokyo
```

Then initialize your shell against that file:

```bash
eval "$(prompto init zsh --config ~/.config/prompto/config.yaml)"
```

## Render a Theme Preview

If you want a quick preview image of the config you are currently using:

```bash
prompto config image --output ./theme-preview.png
```

## Theme Selection Advice

- Pick a `minimal` theme when you do not want icon-heavy glyphs.
- Pick a theme with a clear palette if you plan to reuse colors across many segments.
- Prefer local theme files over generated or remote setups when performance and debuggability matter.

## What to Edit First

Common first edits:

- `prompt` and `rprompt` segment order
- `palette` or `palettes`
- `path`, `git`, `time`, and `session` templates
- `transient` and `rtransient`

The editing model is documented in [Configuration](./configuration.md).
