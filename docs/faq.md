---
title: FAQ
description: Common prompto problems, debugging steps, and operational fixes.
---

## The Prompt Is Slow

Start by measuring, not guessing:

```bash
prompto debug
```

Look for slow segments first.
If the delay only happens inside source control repositories, the SCM segment is usually the main suspect.

For Git-heavy repositories, also check the repository itself:

```bash
git status
git gc
```

If you use daemon rendering, confirm the daemon is healthy:

```bash
prompto daemon status
```

## Icons Render as Rectangles or Empty Boxes

Your terminal font is missing the required glyphs.
Install a Nerd Font and configure the terminal to use it.
See [Fonts](./fonts.md).

## Conda or Python venv Prepends Its Own Prompt

Disable the other tool’s prompt modification so `prompto` is the single source of truth.

### Conda

```bash
conda config --set changeps1 False
```

### Python `venv`

Set this before shell initialization:

```bash
export VIRTUAL_ENV_DISABLE_PROMPT=1
```

PowerShell:

```powershell
$env:VIRTUAL_ENV_DISABLE_PROMPT = 1
```

## Right Prompt or Transient Prompt Does Not Behave as Expected in bash

For `bash`, richer prompt behavior such as right prompt and transient prompt depends on `ble.sh`.
Without it, keep expectations to a conventional left prompt.

## PowerShell Prompt Rendering Looks Wrong After an Upgrade

Check these first:

- terminal encoding
- execution policy fallback
- PowerShell profile load order

If needed, switch init to the `--eval` form documented in [Shell initialization](./shell-init.md).

## I Want to Inspect the Current Config as Parsed by prompto

Export it:

```bash
prompto config export --format yaml
```

This is useful when you suspect a config merge, migration, or formatting issue.

## I Want to Toggle a Segment Without Editing the Config

Use the toggle command:

```bash
prompto toggle git
```

Check the current toggled state:

```bash
prompto get toggles
```

You can also mark a segment with `toggled: true` so it starts disabled by default.

## What Should I Do Before Reporting a Bug?

Collect the basic facts first:

- `prompto version`
- shell and shell version
- terminal emulator
- minimal config that reproduces the issue
- `prompto debug` output if performance is involved
- `prompto daemon status` if daemon rendering is involved
