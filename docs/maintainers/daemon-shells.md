---
title: Daemon — Per-Shell Integration & Debugging
description: How each shell detects vim-mode changes and routes the resulting
  repaint through the daemon's Soft-Cancel path.
---

> **Status:** Skeleton (Phase A3 of the daemon cleanup). Section bodies are
> filled in across Phase E2/E3 as the embedded shell scripts get header
> comments and snapshot tests. Treat any "TBD" below as a real gap, not
> placeholder prose.

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

## Cross-shell summary table

Keep this table in sync with the per-shell sections below.

| Shell | Detection mechanism | Redraw trigger | `--repaint` carrier | ble.sh required? |
|---|---|---|---|---|
| Zsh | `zle-keymap-select` widget (via `_omp_create_widget`) | `zle .reset-prompt` | `_omp_vim_mode_repaint=1` flag read by prompt fn | no |
| Fish | `--on-variable fish_bind_mode` | `commandline -f repaint` | `fish_prompt` reads a flag | no |
| PowerShell | `Set-PSReadLineKeyHandler` on `Escape` / `i` / `a` in vi mode | `[Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()` | `prompt` function reads a flag | no |
| Bash | `ble/keymap` hook (`_prompto_ble_keymap_change`) | `ble.sh` repaint | TBD (E2) | **yes — `ble.sh` must be sourced first** |

## Zsh — `src/shell/scripts/prompto.zsh`

### Detection mechanism

TBD (E2). Reference: function `_prompto_zle-keymap-select` (line 374
in the current script) created via `_prompto_create_widget`.

### Redraw trigger

TBD (E2). Reference: `_prompto_reset_prompt_if_zle` (line 522). Calls
`zle .reset-prompt` when ZLE is active.

### `--repaint` wiring

TBD (E2). Trace the flag from the keymap handler → prompt function →
`_prompto_daemon_render` (line 421). Document the variable name (currently
likely `_omp_vim_mode_repaint` or similar — confirm before filling in).

### Debugging tips

TBD (E2). Suggested content:

- How to verify the daemon sees `repaint=true` (turn on daemon logging via
  `prompto daemon log <path>`; grep for `RenderPrompt.*repaint=true`).
- How to verify `zle .reset-prompt` fired (zsh-side `setopt xtrace`).
- Common failure: the keymap handler runs but the prompt function doesn't
  see the flag — flag was reset by an intermediate hook.

## Fish — `src/shell/scripts/prompto.fish`

### Detection mechanism

TBD (E2). Reference: `function _prompto_on_bind_mode_change --on-variable
fish_bind_mode` (line 416). Fish updates `fish_bind_mode` on every keymap
switch; the event handler fires synchronously.

### Redraw trigger

TBD (E2). Reference: `prompto_repaint_prompt` (line 355) → likely
`commandline -f repaint`. Confirm in script during E2.

### `--repaint` wiring

TBD (E2). Trace from `_prompto_on_bind_mode_change` → `fish_prompt` (line
86) → `_prompto_daemon_render` (line 483). Document the carrier variable.

There is also a `--on-signal USR1` handler (`_prompto_daemon_repaint`, line
444) — explain when that fires and whether the daemon ever sends it.

### Debugging tips

TBD (E2). Suggested content:

- Verify the event handler is registered: `functions _prompto_on_bind_mode_change`.
- Verify the daemon flag is set: enable daemon logging.
- Common failure: `fish_bind_mode` is updated synchronously but the handler
  runs *after* `fish_prompt`, so the first redraw uses stale mode. Workaround:
  re-render in the handler itself (current behavior).

## PowerShell — `src/shell/scripts/prompto.ps1`

### Detection mechanism

TBD (E2). PSReadLine-based; uses `Set-PSReadLineKeyHandler` for `Escape`
in `ViMode` and `i` / `a` / `o` in `ViCommandMode`. The PS1 script is the
longest of the four (880 LoC) and the most pattern-different from the
others — fully document the key bindings here.

### Redraw trigger

TBD (E2). Reference: `[Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()`.

### `--repaint` wiring

TBD (E2). Trace from each key handler → flag variable →
`prompt` function → `prompto render` invocation.

Special concern: in `ViMode` ESC, the handler must also call the native
`ViCommandMode()` method (and vice versa) so that PSReadLine's own state
machine progresses; the `InvokePrompt()` alone is not enough.

### Debugging tips

TBD (E2). Suggested content:

- `Get-PSReadLineKeyHandler -Chord Escape` to inspect what's bound.
- `Set-PSReadLineOption -EditMode Vi` is required upstream of any of this.
- Common failure: another module (e.g. PSFzf) rebinds Escape after prompto's
  init runs. Document init order.

## Bash + ble.sh — `src/shell/scripts/prompto.bash`

### Pre-requisite

`ble.sh` (https://github.com/akinomyoga/ble.sh) must be installed and
sourced **before** prompto's init runs. Native Bash readline does not
expose mode-change hooks to shell scripts, so without `ble.sh` the
vim-toggle feature degrades to "no repaint until the next keystroke."

Detection: prompto checks `$BLE_SESSION_ID` at init time. If unset, the
vim hooks are skipped silently. **Document this loudly** — it's the most
common "why doesn't it work for me" question.

### Detection mechanism

TBD (E2). Reference: `_prompto_ble_keymap_change` (line 256) and
`_prompto_register_vim_hooks` (line 395). Document which ble.sh hook
fires (`bleopt` vs keymap binding).

### Redraw trigger

TBD (E2). Reference: ble.sh's own redraw (`ble/widget/redraw-current-line`
per the plan doc — confirm in script during E2).

### `--repaint` wiring

TBD (E2). Trace from `_prompto_ble_keymap_change` → hook → `_prompto_daemon_render`
(line 313). Bash has the `_prompto_daemon_*` family parallel to the
non-daemon path; document that this duplication is intentional.

### Debugging tips

TBD (E2). Suggested content:

- `echo $BLE_SESSION_ID` — must be non-empty.
- `bleopt | grep prompt` — what's currently set.
- Common failure: ble.sh is sourced *after* prompto init, so
  `BLE_SESSION_ID` is empty at the moment prompto's init reads it. Fix:
  source ble.sh first.

## Adding support for a new shell

If/when a new shell is added, this section becomes the checklist:

1. **Decide:** does this shell expose mode-change hooks to user scripts?
   - Yes → proceed with detection + redraw + repaint wiring.
   - No → vim-mode integration is not supported for this shell. Document
     the limitation in the user-facing docs.
2. **Implement** the three contract steps above.
3. **Add** a Go init helper in `src/shell/<shell>.go` and embed the
   script in `src/shell/scripts/prompto.<ext>`.
4. **Wire** the snapshot test in `src/shell/daemon_scripts_test.go`.
5. **Add a new section to this doc** matching the four-heading template.
6. **Update** the cross-shell summary table at the top.

## See also

- `src/daemon/ARCHITECTURE.md` — daemon-side cancel model and the
  Soft/Hard distinction that `--repaint` activates.
- `.claude/docs/shell-vim-mode-plan.md` — original design plan (historical;
  superseded by this doc and ARCHITECTURE.md, kept for provenance).
