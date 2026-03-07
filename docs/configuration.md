---
title: Prompto Configuration Guide
description: Exhaustive user-facing reference for writing prompto YAML configuration files.
---

## Scope
This guide documents how `prompto` reads and interprets configuration from a user perspective.

It covers:
- File location and format
- Layout structure (`prompt`, `rprompt`, `secondary`, `transient`, `rtransient`)
- Segment definitions and type inference
- Separator style system (`style`, `leading_style`, `trailing_style`)
- Valid style values and glyph pairs
- Async/pending rendering behavior
- Daemon and vim settings
- Validation rules and common errors

## File Location and Format
- Default config path:
  - macOS/Linux: `${XDG_CONFIG_HOME:-$HOME/.config}/prompto/config.yaml`
  - Windows: `%UserConfigDir%/prompto/config.yaml`
- Supported extension: `.yaml` and `.yml`
- Supported format: YAML only

## Mental Model
A config has two layers:
1. Layout lines (`prompt`, `rprompt`, `secondary`, `transient`, `rtransient`) that place named segments.
2. Segment tables (for example `path`, `git.main`, `time.transient`) that define rendering behavior.

Prompt lines reference segment names. Segment names resolve to tables at top level.

## Top-Level Keys
### Core Layout Keys
- `prompt`: array of left-aligned primary prompt lines
- `rprompt`: array of right-aligned primary prompt lines
- `secondary`: array of left-aligned secondary prompt lines
- `transient`: array of left-aligned transient prompt lines
- `rtransient`: array of right-aligned transient prompt lines

Each line entry supports:
- `segments`: required array of segment names
- `filler`: optional filler text used between left/right lines
- `style`: optional separator shortcut
- `leading_style`, `trailing_style`: explicit separator style aliases
- `leading_separator`, `trailing_separator`: explicit separator glyphs

### Metadata Keys
These are consumed from YAML and applied globally:
- `palette`
- `palettes`
- `var`
- `maps`
- `upgrade`
- `cycle`
- `iterm_features`
- `vim-mode`
- `accent_color`
- `daemon_idle_timeout`
- `daemon_timeout`
- `render_pending_icon`
- `render_pending_background`
- `console_title_template`
- `pwd`
- `terminal_background`
- `tooltips_action`
- `async`
- `shell_integration`
- `final_space`
- `patch_pwsh_bleed`
- `enable_cursor_positioning`

### Compatibility Notes
- `version` can exist in your file but is not used by the parser logic.
- Unknown scalar/non-map top-level keys are ignored.
- Unknown map keys may be interpreted as segment tables if they look like one.

## Prompt Line Object Rules
### Separator Fields Are Mutually Exclusive
For each line object:
- You cannot set both `leading_style` and `leading_separator`.
- You cannot set both `trailing_style` and `trailing_separator`.
- You cannot mix `style` with any explicit leading/trailing style/separator fields.

### `style` Shortcut Semantics
`style` on a line is alignment-aware:
- Left-aligned lines (`prompt`, `secondary`, `transient`): `style` sets trailing separator.
- Right-aligned lines (`rprompt`, `rtransient`): `style` sets leading separator.

If you want both outer edges explicitly, set both:
- `leading_style`
- `trailing_style`

## Segment Definitions
A segment table can be defined in two equivalent ways.

### Flat Instance Name
```yaml
git.main:
  template: " ... "
```

### Nested by Type
```yaml
git:
  main:
    template: " ... "
```

Both produce segment name `git.main`.

### Type Inference
`type` is optional when it can be inferred:
- `path:` infers `type: path`
- `git.main:` infers `type: git`
- Nested form under known type (`git: main:`) infers that type

If inference fails, parsing errors with `segment <name> is missing type`.

### Segment Alias
- If `alias` is omitted, alias defaults to the segment table name.
- Toggle and runtime behavior use this resolved alias.

## Segment Fields
Common segment fields include:
- `type`
- `alias`
- `style`
- `template`
- `templates`
- `templates_logic`
- `foreground`, `background`
- `foreground_templates`, `background_templates`
- `leading_style`, `trailing_style`
- `leading_separator`, `trailing_separator`
- `render_pending_icon`, `render_pending_background`
- `options`
- `cache`
- `interactive`
- `timeout`
- `min_width`, `max_width`
- `include_folders`, `exclude_folders`
- `force`
- `toggled`

### `properties` vs `options`
- Use `options`.
- `properties` is accepted for backward compatibility and migrated internally.

## Two Style Systems (Important)
### 1. Separator Alias Styles
These define separator glyph pairs and are valid for:
- Prompt line `style`
- Prompt line `leading_style` and `trailing_style`
- Segment `leading_style` and `trailing_style`
- Segment `style` when using separator-shortcut behavior

Valid separator alias values:
- `powerline`
- `powerline_thin`
- `rounded`
- `rounded_thin`
- `slant`
- `block`
- `flame`
- `pixel`
- `lego`

Glyph pairs (`leading` / `trailing`):
- `powerline`: `` / ``
- `powerline_thin`: `` / ``
- `rounded`: `` / ``
- `rounded_thin`: `` / ``
- `slant`: `` / ``
- `block`: `` / ``
- `flame`: `` / ``
- `pixel`: `` / ``
- `lego`: `` / ``

### 2. Segment Render Styles
Segment engine render styles are:
- `plain`
- `powerline`
- `accordion`
- `diamond`

In layout mode, if segment `style` is one of separator aliases above,
it is treated as separator shorthand and normalized to `diamond` with resolved separators.

## Pending Rendering (Async)
When async rendering is used, slow segments can render as pending:
- Cached value if available, otherwise `...`
- Prefixed with pending icon
- Optional pending background override

Resolution order:
1. Segment-level `render_pending_icon`
2. Global `render_pending_icon`
3. Default icon: `\uf254 ` (hourglass)

Background resolution order:
1. Segment-level `render_pending_background`
2. Global `render_pending_background`
3. No override

## Daemon Timing Controls
- `daemon_timeout` (milliseconds):
  - Time budget before first response returns partial prompt
  - Default: `100`
- `daemon_idle_timeout` (minutes as string):
  - Idle shutdown timeout after tracked sessions end
  - Default: `"5"`
  - `"none"` disables idle shutdown

## Vim Mode
Use top-level `vim-mode`:
```yaml
vim-mode:
  enabled: true
  cursor_shape: true
  cursor_blink: false
```

Rejected legacy form:
- Top-level `vim:` for these settings is invalid

## Layout Naming Rules
- Segment names in `segments:` must exist as segment tables.
- Missing references fail parse (`unknown segment` error).
- Duplicate segment instance names fail parse (`duplicate segment instance`).

## Rejected / Deprecated Top-Level Keys
The parser rejects these aliases:
- `secondary_prompt` (use `secondary`)
- `transient_prompt` (use `transient`)
- `transient_rprompt` (use `rtransient`)

## Minimal Example
```yaml
final_space: true
render_pending_background: darkGray

prompt:
  - segments: [session, path]

rprompt:
  - segments: [git, vim]

transient:
  - segments: [path]

rtransient:
  - leading_style: rounded
    trailing_style: rounded
    segments: [git.transient, time.transient]

session:
  style: powerline
  foreground: "#ffff00"
  background: "#000000"
  template: "{{ if .SSHSession }} {{ .UserName }}@{{ .HostName }} {{ end }}"

path:
  style: powerline
  foreground: "#ffffff"
  background: blue
  template: " {{ .Path }} "
  options:
    style: powerlevel

# Implicit type: git
# Name is git
# Uses full status

git:
  style: powerline
  foreground: "#3a3a3a"
  background_templates:
    - "{{ if or (.Working.Changed) (.Staging.Changed) }}yellow{{ else }}green{{ end }}"
  template: " ... "
  options:
    fetch_status: true

# Implicit type: git from git.* naming

git.transient:
  style: powerline
  foreground: "#3a3a3a"
  background_templates:
    - "{{ if or (.Working.Changed) (.Staging.Changed) }}yellow{{ else }}green{{ end }}"
  template: " ... "
  options:
    fetch_status: true

# Implicit type: time

time.transient:
  style: powerline
  foreground: "#ffffff"
  background: "#6a6a6a"
  template: " {{ .CurrentDate | date .Format }} "
  options:
    time_format: "15:04"

vim:
  style: powerline
  foreground: "#ffffff"
  background: "#ff5f57"
  template: "{{ if .Normal }} NORMAL {{ end }}"
```

## Supported Segment Types
Current segment `type` values:

```text
angular, argocd, aurelia, aws, az, azd, azfunc, battery, bazel, brewfather, buf, bun,
carbonintensity, cds, cf, cftarget, clojure, cmake, connection, copilot, crystal, dart,
deno, docker, dotnet, elixir, executiontime, exit, firebase, flutter, fortran, fossil, gcp,
git, gitversion, go, haskell, helm, ipify, java, http, jujutsu, julia, kotlin, kubectl,
lastfm, lua, mercurial, mojo, mvn, nba, nbgv, nightscout, nim, nix-shell, node, npm, nx,
ocaml, os, owm, path, perl, php, plastic, pnpm, project, pulumi, python, quasar, r,
react, root, ruby, rust, sapling, session, shell, sitecore, spotify, status, strava,
svelte, svn, swift, sysinfo, talosctl, tauri, terraform, text, time, todoist, ui5tooling,
umbraco, unity, upgrade, vim, v, vala, wakatime, winget, winreg, withings, xmake, yarn,
ytm, zig
```

## Practical Recommendations
1. Use explicit `leading_style` and `trailing_style` when you need exact shape control.
2. Keep segment names stable (`git`, `git.transient`, `time.transient`) for reusable layouts.
3. Put long template logic in one place and reuse via multiple prompt lines.
4. Use `daemon_timeout` to tune first paint responsiveness.
5. Use `render_pending_*` to make async behavior visually intentional.
