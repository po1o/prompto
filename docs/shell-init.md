---
title: Shell Initialization
description: Enable prompto in supported shells and point it at your local configuration.
---

## Supported Shells

The current `init` command supports:

- `zsh`
- `bash`
- `fish`
- `powershell`
- `pwsh`

Check what shell you are currently using:

```bash
prompto shell
```

## General Pattern

Initialization always starts from:

```bash
prompto init <shell>
```

When you want to use a specific config file, pass a local path with `--config`:

```bash
prompto init zsh --config ~/.config/prompto/config.yaml
```

This fork documents local YAML configs as the authoritative path.

## zsh

Add this near the end of `~/.zshrc`:

```bash
eval "$(prompto init zsh --config ~/.config/prompto/config.yaml)"
```

Reload the shell:

```bash
exec zsh
```

## bash

Add this near the end of `~/.bashrc` or `~/.bash_profile`:

```bash
eval "$(prompto init bash --config ~/.config/prompto/config.yaml)"
```

Reload the shell:

```bash
exec bash
```

### Bash right prompt and transient prompt

`bash` supports richer prompt behavior when used with `ble.sh`.
If you only need a primary prompt, the normal init snippet is enough.

## fish

Add this to `~/.config/fish/config.fish`:

```fish
prompto init fish --config ~/.config/prompto/config.yaml | source
```

Reload the shell:

```fish
exec fish
```

## PowerShell

Open your profile:

```powershell
notepad $PROFILE
```

Create it first when necessary:

```powershell
New-Item -Path $PROFILE -Type File -Force
```

Add this near the end:

```powershell
prompto init pwsh --config ~/.config/prompto/config.yaml | Invoke-Expression
```

Reload the profile:

```powershell
. $PROFILE
```

### Execution policy fallback

If local script execution is restricted, use `--eval`:

```powershell
prompto init pwsh --config ~/.config/prompto/config.yaml --eval | Invoke-Expression
```

`--eval` is slower because it emits the full script instead of reusing the cached generated file.

## Useful Flags

- `--config <path>`: use a specific local config file.
- `--print`: print the init script instead of returning the wrapper command.
- `--debug`: write the generated script and print init diagnostics.
- `--strict`: prefer a path that resolves through the executable name instead of the full path.
- `--eval`: PowerShell-specific fallback for stricter execution environments.

## Verifying Initialization

After reloading your shell, confirm that `prompto` is being used:

```bash
prompto version
prompto debug
```

If you are using daemon rendering, you can also check:

```bash
prompto daemon status
```
