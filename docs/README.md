# Documentation

## Overview

This repository keeps the user documentation directly in `docs/`.
These Markdown files are the source of truth for this fork.

## Placeholder Notice

Any remaining `prompto.dev` links in older upstream text are legacy placeholders.
Do not treat them as working documentation for this fork.
Use the Markdown files in this repository instead.

## Start Here

- [Installation](./installation.md): install a binary or build from source.
- [Shell initialization](./shell-init.md): enable `prompto` in `zsh`, `bash`, `fish`, or PowerShell.
- [Fonts](./fonts.md): install and configure a Nerd Font.
- [Themes](./themes.md): browse the bundled themes and copy one into your local config.
- [Configuration](./configuration.md): understand the YAML model.
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
prompt:
  - segments: [path]

path:
  foreground: white
  background: blue
  template: " {{ .Path }} "
```

1. Initialize your shell:

```bash
prompto init
```

`prompto init` detects the current shell automatically.
If you want to generate init code for a specific shell, pass it explicitly.
See [Shell initialization](./shell-init.md) for the exact profile snippets.

## Configuration Model in One Minute

A `prompto` config has two layers:

- Layout lines such as `prompt`, `rprompt`, `transient`, and `rtransient`.
- Named segment definitions at the top level, such as `path`, `git`, or `git.transient`.

A layout line decides where segments appear.
A segment definition decides how one segment renders.

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

The full model is documented in [Configuration](./configuration.md).

## Useful Commands

- Detect the current shell:

```bash
prompto shell
```

- Edit the active config with `$EDITOR`:

```bash
prompto config edit
```

- List bundled themes:

```bash
prompto config list
```

- Write a bundled theme to the default config path:

```bash
prompto config set tokyo
```

- Render a preview image of the active config:

```bash
prompto config image --output ./prompto-preview.png
```

- Inspect prompt timing:

```bash
prompto debug
```

- Check daemon state:

```bash
prompto daemon status
```
