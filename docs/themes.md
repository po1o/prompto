---
title: Themes
description: Use the bundled themes as starting points for your own local prompto configuration.
---

## Where Themes Live

Bundled themes are stored in [`themes/`](../themes).
They are plain YAML files and are meant to be copied, edited, and adapted.

Examples:

- [`themes/agnoster.minimal.prompto.yaml`](../themes/agnoster.minimal.prompto.yaml)
- [`themes/tokyo.prompto.yaml`](../themes/tokyo.prompto.yaml)
- [`themes/powerlevel10k_modern.prompto.yaml`](../themes/powerlevel10k_modern.prompto.yaml)

## Recommended Workflow

1. Pick a theme that is visually close to what you want.
2. Copy it to your local config path.
3. Point shell init at that local file.
4. Edit the copy instead of editing the theme in-place.

Example:

```bash
cp themes/tokyo.prompto.yaml ~/.config/prompto/config.yaml
```

Then initialize your shell against that file:

```bash
eval "$(prompto init zsh --config ~/.config/prompto/config.yaml)"
```

## Export the Active Config

If you want a canonical YAML snapshot of the config you are currently using:

```bash
prompto config export --output ~/.config/prompto/config.yaml
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
