---
title: Daemon — Per-Shell Integration & Debugging
description: How each shell detects vim-mode changes and routes the resulting
  repaint through the daemon's Soft-Cancel path.
---

## Audience

Contributors modifying:

- `src/shell/scripts/prompto.{bash,fish,ps1,zsh}` (the embedded shell scripts)
- `src/shell/{init,bash,fish,pwsh,zsh}.go` (the Go-side init helpers)
- Anything in `src/daemon/` that crosses the shell boundary
  (`PromptRequest.repaint`, `--repaint` CLI flag, daemon-side cancel kind)

If you only consume the prompt from a shell, this doc is not for you — see
the user docs.

## The contract every shell implements

Each supported shell, on every prompt redraw caused by a vim-mode toggle,
must:

1. **Detect** that the mode changed (insert ↔ command / normal ↔ visual).
2. **Trigger** a prompt redraw using the shell's own repaint mechanism.
3. **Wire** the `--repaint` flag into the `prompto render` invocation that
   the redraw triggers — this is what tells the daemon to take the
   Soft-Cancel path (`CancelKind = CancelSoft`, see
   `src/daemon/ARCHITECTURE.md`).

Without step 3, the daemon defaults to Hard-Cancel: it kills the in-flight
git/k8s/etc. computation and starts fresh. Vim toggles then feel laggy.

## Cross-shell summary

| Shell | Detection mechanism | Redraw trigger | `--repaint` carrier | ble.sh required? |
|---|---|---|---|---|
| Zsh | `_prompto_zle-keymap-select` (decorating ZLE's `zle-keymap-select` widget) | `zle reset-prompt` | `_prompto_vim_mode_repaint=1` shell variable | no |
| Fish | `_prompto_on_bind_mode_change --on-variable fish_bind_mode` | `commandline -f repaint` | `_prompto_vim_mode_repaint 1` global variable | no |
| PowerShell | `Set-PSReadLineKeyHandler -ViMode Command/Insert` on the mode-switch keys | `[Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()` | `$script:VimModeRepaint = $true` | no |
| Bash | `_prompto_ble_keymap_change` via ble.sh keymap hook | ble.sh repaint (`ble/widget/...`) | `_prompto_vim_mode_repaint=1` shell variable | **yes — `ble.sh` must be sourced first** |

A snapshot test in `src/shell/daemon_scripts_test.go`
(`TestDaemonScriptsWireRepaintAndModeDetection`) locks the `--repaint` +
mode-detection contract per shell. Changing any of the cells above must
update both this table and that test.

## Zsh — `src/shell/scripts/prompto.zsh`

### Detection mechanism

`_prompto_zle-keymap-select` is decorated onto ZLE's built-in
`zle-keymap-select` widget via `_prompto_create_widget`. The widget fires on
every `$KEYMAP` change (vicmd ↔ main), giving us a synchronous,
zero-polling hook.

### Redraw trigger

`zle reset-prompt` (from within the keymap-select hook). It re-renders the
prompt without restarting any in-flight shell command.

### `--repaint` wiring

1. Keymap hook sets `_prompto_vim_mode_repaint=1`.
2. `zle reset-prompt` runs.
3. `_prompto_daemon_render` reads the flag, appends `--repaint` to the
   prompto invocation, then resets the flag.
4. Daemon takes the Soft-Cancel path (preserves in-flight computations).

### Debugging tips

- Verify the widget is decorated: `zle -l zle-keymap-select | grep prompto`.
- Verify the daemon sees `repaint=true`: enable daemon logging with
  `prompto daemon log <path>` and grep for `repaint=true`.
- Common failure: another plugin redecorates `zle-keymap-select` after
  prompto's init runs — check order in `.zshrc`.

## Fish — `src/shell/scripts/prompto.fish`

### Detection mechanism

`function _prompto_on_bind_mode_change --on-variable fish_bind_mode` runs
whenever fish updates the bind-mode variable on ESC/i.

### Redraw trigger

`commandline -f repaint` from the handler.

### `--repaint` wiring

1. Handler sets `_prompto_vim_mode_repaint 1` and triggers
   `commandline -f repaint`.
2. `fish_prompt` runs, which calls `_prompto_daemon_render`.
3. `_prompto_daemon_render` reads the flag, appends `--repaint`, then resets.
4. Daemon takes the Soft-Cancel path.

There is also a `_prompto_daemon_repaint --on-signal USR1` handler. This is
a defensive repaint trigger; the daemon does not currently send USR1, so
the handler is dormant. Documented here so it isn't mistaken for dead code.

### Debugging tips

- Verify the handler is registered: `functions _prompto_on_bind_mode_change`.
- Verify daemon flag: enable daemon logging and grep for `repaint=true`.
- Common failure: `fish_bind_mode` updates synchronously but the handler
  runs *after* `fish_prompt`, so the first redraw after init may use stale
  mode. The handler triggers a re-render to compensate.

## PowerShell — `src/shell/scripts/prompto.ps1`

### Detection mechanism

`Set-PSReadLineKeyHandler -ViMode Command/Insert` bindings on the keys that
toggle vim modes (Escape, i, a, Enter in command mode, etc.). Requires
`Set-PSReadLineOption -EditMode Vi` upstream.

### Redraw trigger

`[Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()` from inside the
key handler. Some handlers also call the native PSReadLine method (e.g.
`ViCommandMode()`) so PSReadLine's own state machine progresses.

### `--repaint` wiring

1. Key handler sets `$script:VimModeRepaint = $true`.
2. Handler calls `InvokePrompt()`, which re-runs the `prompt` function.
3. `prompt` reads `$script:VimModeRepaint`, appends `--repaint` to the
   prompto invocation, then clears the flag.
4. Daemon takes the Soft-Cancel path.

### Debugging tips

- List active key handlers: `Get-PSReadLineKeyHandler -Chord Escape`.
- `Set-PSReadLineOption -EditMode Vi` is a prerequisite — without it the
  vim-mode handlers are never reachable.
- Common failure: another module (e.g. PSFzf, Terminal-Icons) rebinds keys
  after prompto's init runs. Document init order in `$PROFILE`.

## Bash + ble.sh — `src/shell/scripts/prompto.bash`

### Pre-requisite

`ble.sh` (<https://github.com/akinomyoga/ble.sh>) must be installed and
sourced **before** prompto's init runs. Native Bash readline does not
expose mode-change hooks to shell scripts, so without `ble.sh` the
vim-toggle feature degrades to "no repaint until the next keystroke."

Detection: prompto checks `$BLE_SESSION_ID` at init time. If unset, the
vim hooks are skipped silently.

### Detection mechanism

`_prompto_ble_keymap_change` is registered via the ble.sh keymap hook (see
`_prompto_register_vim_hooks`). ble.sh fires the hook on every keymap
transition.

### Redraw trigger

ble.sh's own repaint mechanism (via `bleopt`-managed prompt rendering).
prompto does not manually trigger a redraw — setting the flag is enough,
because ble.sh repaints the prompt as part of its keymap-change handling.

### `--repaint` wiring

1. ble.sh keymap hook fires `_prompto_ble_keymap_change`.
2. Hook sets `_prompto_vim_mode_repaint=1`.
3. ble.sh repaints the prompt, which invokes `_prompto_daemon_render`.
4. `_prompto_daemon_render` reads the flag, appends `--repaint`, then
   resets.
5. Daemon takes the Soft-Cancel path.

### Debugging tips

- `echo $BLE_SESSION_ID` — must be non-empty for vim hooks to be active.
- `bleopt | grep prompt` — what's currently set for prompt handling.
- Common failure: ble.sh sourced *after* prompto init, so
  `BLE_SESSION_ID` is empty when prompto registers hooks. Fix: source
  ble.sh first.

## Adding support for a new shell

1. **Decide:** does this shell expose mode-change hooks to user scripts?
   - Yes → proceed with detection + redraw + repaint wiring.
   - No → vim-mode integration is not supported. Document the limitation
     in the user-facing docs.
2. **Implement** the three contract steps above (detect, redraw, append
   `--repaint`).
3. **Add** a Go init helper in `src/shell/<shell>.go` and embed the
   script in `src/shell/scripts/prompto.<ext>`.
4. **Add** a row to the cross-shell summary table above.
5. **Add** a case in `daemon_scripts_test.go` — both the existing
   `TestDaemonScriptsIncludePIDAndVimModeSupport` and the
   `TestDaemonScriptsWireRepaintAndModeDetection` snapshot tests must cover
   the new shell.
6. **Add a new section** to this doc matching the four-heading template.

## See also

- `src/daemon/ARCHITECTURE.md` — daemon-side cancel model and the
  Soft/Hard distinction that `--repaint` activates.
- `.claude/docs/shell-vim-mode-plan.md` — original design plan
  (historical; superseded by this doc and ARCHITECTURE.md, kept for
  provenance).
