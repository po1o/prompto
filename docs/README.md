---
title: Documentation
description: GitHub-native documentation for installing, configuring, and operating prompto.
---

## Overview

This repository keeps the user documentation directly in `docs/`.
The Markdown files here are the canonical docs for this fork.

## Placeholder Notice

Any remaining `prompto.dev` references in older upstream text are legacy placeholders and should not be treated as live
documentation links for this fork. Use the Markdown files in this `docs/` tree as the source of truth.

## Start Here

- [Installation](./installation.md): install a binary or build from source.
- [Shell initialization](./shell-init.md): enable `prompto` in `zsh`, `bash`, `fish`, or PowerShell.
- [Fonts](./fonts.md): install and configure a Nerd Font.
- [Themes](./themes.md): use the bundled themes as starting points.
- [Configuration](./configuration.md): write and understand `prompto` YAML configs.
- [Segment reference](./segments/README.md): per-segment docs grouped by category.
- [FAQ](./faq.md): common operational issues and fixes.

## Quick Start

1. Install `prompto`.
2. Create a config at the default location:

```text
macOS/Linux: ${XDG_CONFIG_HOME:-$HOME/.config}/prompto/config.yaml
Windows: %UserConfigDir%/prompto/config.yaml
```

1. Start from a minimal config:

```yaml
final_space: true

prompt:
  - segments: [path]

path:
  foreground: white
  background: blue
  template: " {{ .Path }} "
```

1. Initialize your shell:

```bash
prompto init zsh
prompto init bash
prompto init fish
prompto init pwsh
```

See [Shell initialization](./shell-init.md) for the exact profile snippets.

## Configuration Model in One Minute

A `prompto` config has two layers:

- Prompt layout lines such as `prompt`, `rprompt`, `transient`, and `rtransient`.
- Named segment definitions at the top level, such as `path`, `git`, or `git.transient`.

A layout line places named segments. A segment definition decides how one segment renders.

```yaml
prompt:
  - segments: [session, path]

rprompt:
  - segments: [git]

session:
  foreground: black
  background: yellow
  template: " {{ .UserName }} "

path:
  foreground: white
  background: blue
  template: " {{ .Path }} "

git:
  foreground: black
  background: green
  template: " {{ .HEAD }} "
```

The complete model is documented in [Configuration](./configuration.md).

## Useful Commands

- Detect the current shell:

```bash
prompto get shell
```

- Edit your config with `$EDITOR`:

```bash
prompto config edit
```

- Export the active config as YAML:

```bash
prompto config export --format yaml
```

- Inspect prompt timing:

```bash
prompto debug
```

- Check daemon state:

```bash
prompto daemon status
```
