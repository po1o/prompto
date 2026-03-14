# Shell Initialization

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

The simplest form is:

```bash
prompto init
```

When you run `prompto init` inside a supported shell, prompto detects that shell automatically.

You can still be explicit when you want to generate init code for a specific shell:

```bash
prompto init zsh
```

When you want to use a specific config file, pass it with `--config`:

```bash
prompto init --config ~/.config/prompto/config.yaml
```

## zsh

Add this near the end of `~/.zshrc`:

```bash
eval "$(prompto init --config ~/.config/prompto/config.yaml)"
```

Reload the shell:

```bash
exec zsh
```

## bash

Add this near the end of `~/.bashrc` or `~/.bash_profile`:

```bash
eval "$(prompto init --config ~/.config/prompto/config.yaml)"
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
prompto init --config ~/.config/prompto/config.yaml | source
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
prompto init --config ~/.config/prompto/config.yaml | Invoke-Expression
```

Reload the profile:

```powershell
. $PROFILE
```

### Execution policy fallback

If local script execution is restricted, use `--eval`:

```powershell
prompto init --config ~/.config/prompto/config.yaml --eval | Invoke-Expression
```

`--eval` is slower because it emits the full script instead of reusing the cached generated file.

## Useful Flags

- `--config <path>`: use a specific config file.
- `--print`: print the init script instead of the wrapper command.
- `--debug`: print init diagnostics and keep the generated script visible.
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
