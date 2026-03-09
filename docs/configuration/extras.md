---
title: Extras and Shell Features
description: Configure transient prompts, tooltips, vim mode, upgrade behavior, and shell extras.
---

## Scope

This page covers the top-level features that are not ordinary layout segment tables.

## Transient Prompt

`transient` and `rtransient` define the prompt that replaces the old primary prompt after you press Enter.

```yaml
transient:
  - segments: [path.transient]

rtransient:
  - segments: [git.transient, time.transient]
```

Important behavior:

- transient prompt content is streamed together with primary updates
- if a transient segment is still pending when you press Enter, the pending rendering is what gets printed
- the old transient line is not updated afterward because the next prompt cancels the previous render generation

### Shell support

Current shell integration supports transient prompts in:

- `zsh`
- `fish`
- `pwsh` / `powershell`
- `bash` when used with `ble.sh` for the richer prompt features

## Secondary Prompt

`secondary` defines the continuation prompt shown for multi-line command entry.

```yaml
secondary:
  - segments: [text.secondary]

text.secondary:
  type: text
  foreground: darkGray
  background: transparent
  template: " > "
```

## Debug Prompt

`debug_prompt` is a special extra segment used for debug contexts.

```yaml
debug_prompt:
  type: text
  foreground: white
  background: red
  template: " [DBG] "
```

## Valid and Error Line

`valid_line` and `error_line` define the prompt suffix used by shell integrations that support line validity feedback.

```yaml
valid_line:
  type: text
  foreground: white
  background: green
  template: " ✔ "

error_line:
  type: text
  foreground: white
  background: red
  template: " ✘ "
```

### Shell support

This behavior is PowerShell-specific.

## Tooltips

`tooltips` is a list of inline segment definitions that render while typing specific commands.
They are not named through `prompt` layout arrays.
Each tooltip segment includes its own `tips` list.

```yaml
tooltips_action: extend
tooltips:
  - type: git
    tips: [git, g]
    foreground: black
    background: yellow
    template: " {{ .HEAD }} {{ .Working.String }} "
    options:
      fetch_status: true
  - type: aws
    tips: [aws, terraform]
    foreground: black
    background: blue
    template: " {{ .Profile }}{{ if .Region }}@{{ .Region }}{{ end }} "
```

### `tooltips_action`

Accepted values:

- `replace`
- `extend`
- `prepend`

If omitted, the effective default behavior is `replace`.

### Shell support

Tooltips are supported by the current shell integrations for:

- `zsh`
- `fish`
- `pwsh` / `powershell`

## Console Title

Use `console_title_template` to control the terminal title.

```yaml
console_title_template: "{{ .Folder }}{{ if .Root }} :: root{{ end }} :: {{ .Shell }}"
```

## Vim Mode

Use the top-level `vim-mode` table.

```yaml
vim-mode:
  enabled: true
  cursor_shape: true
  cursor_blink: false
```

Fields:

- `enabled`: enable vim mode integration
- `cursor_shape`: change cursor shape based on vim mode when the shell supports it
- `cursor_blink`: change cursor blink behavior when the shell supports it

This is the supported form.
Do not use an old top-level `vim:` table for these settings.

## Upgrade Settings

`upgrade` controls update notices and automatic upgrade checks.

```yaml
upgrade:
  notice: true
  auto: false
  interval: 168h
  source: github
```

Fields:

- `notice`: show an upgrade notice
- `auto`: perform automatic upgrades
- `interval`: Go duration string such as `24h`, `168h`, or `30m`
- `source`: `cdn` or `github`

Notes:

- automatic upgrade will not jump major versions without explicit intent
- upgrade features are suppressed when async shell loading is enabled

## Daemon Timing

These settings affect daemon-mode prompt rendering.

```yaml
daemon_timeout: 100
daemon_idle_timeout: "5"
render_pending_icon: " "
render_pending_background: darkGray
```

### `daemon_timeout`

Milliseconds to wait before returning the initial batch and continuing with streamed updates.
Default: `100`

### `daemon_idle_timeout`

Minutes, stored as a string.
Examples:

- `"5"`
- `"30"`
- `"none"`

`none` disables idle shutdown.

### Pending rendering defaults

`render_pending_icon` and `render_pending_background` are the global defaults used when a slow segment is shown as
pending.

## Maps

Use `maps` to rewrite usernames, hostnames, and shell names before they enter templates.

```yaml
maps:
  user_name:
    polo: work
  host_name:
    gally: laptop
  shell_name:
    pwsh: PowerShell
```

## Shell Integration Flags

### `shell_integration`

Enables FinalTerm-style shell integration sequences when the shell supports them.

### `enable_cursor_positioning`

Allows `bash` and `zsh` integrations to query cursor position so newline handling can be smarter.
Use this only when you need it.

### `patch_pwsh_bleed`

Applies the PowerShell background-bleed workaround in supported contexts.
Only relevant to PowerShell.

### `iterm_features`

Available values:

- `prompt_mark`
- `current_dir`
- `remote_host`

Example:

```yaml
iterm_features:
  - prompt_mark
  - current_dir
```

`prompt_mark` is only emitted for supported shells.

## Miscellaneous Top-Level Settings

### `cursor_padding`

Add one space between the rendered left prompt and the cursor.


### `pwd`

Emit terminal working-directory integration.
Accepted values are terminal-specific sequences such as:

- `osc7`
- `osc51`
- `osc99`

This field can also be templated.

### `terminal_background`

Declare your terminal background color so color composition behaves more predictably in some terminals.

### `accent_color`

Fallback value used when the `accent` keyword is requested but the platform cannot provide an accent color.

### `async`

Enable async shell loading for supported shells.
This is separate from daemon streaming.
