---
title: Installation
description: Install prompto from a release or from source, then set up your shell.
---

## Scope

This page covers installing the `prompto` executable itself.
After that, continue with [Shell initialization](./shell-init.md), [Fonts](./fonts.md), and
[Configuration](./configuration.md).

## Option 1: Download a Release

The simplest install path is to download a prebuilt binary from the GitHub releases page:

- [Latest release](https://github.com/po1o/prompto/releases/latest)

Place the executable somewhere on your `PATH`, for example:

- macOS/Linux: `~/bin`, `~/.local/bin`, or `~/opt/go/bin`
- Windows: a directory already present in `PATH`

Verify the install:

```bash
prompto version
```

## Option 2: Build from Source

This repository keeps the Go module in `src/`.
From the repository root:

```bash
cd src
go build -o "$HOME/opt/go/bin/prompto" .
```

Or install it into your Go binary directory:

```bash
cd src
go install .
```

Then verify:

```bash
prompto version
```

## Upgrade Later

Once `prompto` is installed in a writable location, you can upgrade with:

```bash
prompto upgrade
```

You can also enable the notice or automatic upgrade behavior in config:

```yaml
upgrade:
  notice: true
  auto: false
  interval: 168h
  source: github
```

The upgrade settings are documented in [Configuration extras](./configuration/extras.md).

## Default Config Location

If you do not pass `--config`, `prompto` looks for:

```text
macOS/Linux: ${XDG_CONFIG_HOME:-$HOME/.config}/prompto/config.yaml
Windows: %UserConfigDir%/prompto/config.yaml
```

Create the directory if it does not exist.

## Next Steps

1. Follow [Shell initialization](./shell-init.md).
2. Install a compatible font from [Fonts](./fonts.md).
3. Start from a local theme or your own file with [Themes](./themes.md).
4. Build your config with [Configuration](./configuration.md).
