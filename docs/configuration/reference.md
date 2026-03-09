---
title: Configuration Reference
description: Exhaustive field reference for prompto YAML configs and supported segment types.
---

## File Format

- Supported format: YAML only
- Supported extensions: `.yaml`, `.yml`
- Default path:
  - macOS/Linux: `${XDG_CONFIG_HOME:-$HOME/.config}/prompto/config.yaml`
  - Windows: `%UserConfigDir%/prompto/config.yaml`

## Compatibility Notes

- `version` may exist in your file, but it is not used by the current parser logic.
- Unknown scalar top-level keys are ignored.
- Unknown map-shaped top-level keys may be interpreted as segment tables.
- Legacy aliases `secondary_prompt`, `transient_prompt`, and `transient_rprompt` are rejected.

## Top-Level Keys

| Key | Type | Meaning |
| --- | --- | --- |
| `prompt` | `[]PromptLayout` | left-aligned primary prompt lines |
| `rprompt` | `[]PromptLayout` | right-aligned primary prompt lines |
| `secondary` | `[]PromptLayout` | continuation prompt lines |
| `transient` | `[]PromptLayout` | left transient prompt lines |
| `rtransient` | `[]PromptLayout` | right transient prompt lines |
| `palette` | `map[string]string` | base palette values |
| `palettes` | object | named palettes selected by template |
| `var` | map | user-defined values for templates |
| `maps` | object | user, host, and shell text remapping |
| `upgrade` | object | upgrade notice and auto-upgrade settings |
| `cycle` | `[]Set` | repeating foreground/background pairs |
| `iterm_features` | `[]string` | iTerm-specific shell integration features |
| `vim-mode` | object | vim mode and cursor settings |
| `accent_color` | color | accent fallback color |
| `daemon_idle_timeout` | string | daemon idle shutdown timeout in minutes or `none` |
| `daemon_timeout` | int | initial daemon wait in milliseconds |
| `render_pending_icon` | string | global pending icon |
| `render_pending_background` | color | global pending background |
| `console_title_template` | string | terminal title template |
| `pwd` | string | working-directory OSC integration mode |
| `terminal_background` | color | declared terminal background color |
| `tooltips_action` | string | `replace`, `extend`, or `prepend` |
| `tooltips` | `[]Segment` | tooltip segment definitions |
| `debug_prompt` | `Segment` | debug prompt segment |
| `valid_line` | `Segment` | prompt suffix for valid input |
| `error_line` | `Segment` | prompt suffix for invalid input |
| `async` | bool | enable async shell loading |
| `shell_integration` | bool | enable shell integration sequences |
| `cursor_padding` | bool | add one space between the left prompt and the cursor |
| `patch_pwsh_bleed` | bool | PowerShell background-bleed workaround |
| `enable_cursor_positioning` | bool | allow cursor position queries |

## Prompt Layout Fields

| Field | Type | Meaning |
| --- | --- | --- |
| `segments` | `[]string` | segment names used in this line |
| `filler` | string | repeated fill text between left and right content |
| `style` | string | separator alias shortcut |
| `leading_style` | string | explicit leading separator alias |
| `trailing_style` | string | explicit trailing separator alias |
| `leading_separator` | string | explicit leading separator glyph |
| `trailing_separator` | string | explicit trailing separator glyph |

### Layout Validation Rules

- `style` cannot be combined with explicit leading or trailing style or separator fields.
- `leading_style` and `leading_separator` are mutually exclusive.
- `trailing_style` and `trailing_separator` are mutually exclusive.
- `leading_diamond` and `trailing_diamond` are not allowed in layout YAML input.

## Segment Fields

| Field | Type | Meaning |
| --- | --- | --- |
| `type` | string | segment type |
| `alias` | string | runtime alias used for toggles and references |
| `style` | string | segment render style or separator shortcut |
| `template` | string | single template for rendered text |
| `templates` | `[]string` | template list |
| `templates_logic` | string | `join` or `first_match` |
| `foreground` | color | base foreground color |
| `background` | color | base background color |
| `foreground_templates` | `[]string` | conditional foreground templates |
| `background_templates` | `[]string` | conditional background templates |
| `leading_style` | string | leading separator alias for the segment |
| `trailing_style` | string | trailing separator alias for the segment |
| `leading_separator` | string | leading separator glyph |
| `trailing_separator` | string | trailing separator glyph |
| `leading_diamond` | string | explicit leading diamond glyph |
| `trailing_diamond` | string | explicit trailing diamond glyph |
| `render_pending_icon` | string | per-segment pending icon override |
| `render_pending_background` | color | per-segment pending background override |
| `options` | map | segment-specific options |
| `cache` | object | segment cache config |
| `interactive` | bool | interactive terminal-writer mode |
| `timeout` | int | timeout in milliseconds |
| `min_width` | int | minimum terminal width |
| `max_width` | int | maximum terminal width |
| `include_folders` | `[]string` | anchored regex allow-list for current directory |
| `exclude_folders` | `[]string` | anchored regex deny-list for current directory |
| `force` | bool | render even if the text would be empty |
| `toggled` | bool | start disabled in the toggle cache |
| `tips` | `[]string` | tooltip trigger words |
| `newline` | bool | segment requests a newline |

## Segment Render Styles

These are the actual render styles used by the segment engine:

- `plain`
- `powerline`
- `accordion`
- `diamond`

If a segment `style` is one of the separator aliases below, layout normalization treats it as shorthand and resolves
it into `diamond` with concrete separator glyphs.

## Separator Aliases

| Alias | Leading glyph | Trailing glyph |
| --- | --- | --- |
| `powerline` | `` | `` |
| `powerline_thin` | `` | `` |
| `rounded` | `` | `` |
| `rounded_thin` | `` | `` |
| `slant` | `` | `` |
| `block` | `` | `` |
| `flame` | `` | `` |
| `pixel` | `` | `` |
| `lego` | `` | `` |

## Cache Object

```yaml
cache:
  duration: 30s
  strategy: folder
```

### Fields

| Field | Type | Meaning |
| --- | --- | --- |
| `duration` | duration string | cache lifetime |
| `strategy` | string | `folder`, `session`, or `device` |

## Palettes Object

```yaml
palettes:
  template: "{{ if eq .Shell \"pwsh\" }}windows{{ else }}unix{{ end }}"
  list:
    unix:
      fg: "#e6edf3"
    windows:
      fg: "#ffffff"
```

### Fields

| Field | Type | Meaning |
| --- | --- | --- |
| `template` | string | template resolving to the palette name |
| `list` | map | named palette definitions |

## Maps Object

| Field | Meaning |
| --- | --- |
| `user_name` | rewrite user names before templates see them |
| `host_name` | rewrite host names before templates see them |
| `shell_name` | rewrite shell names before templates see them |

## Upgrade Object

| Field | Type | Meaning |
| --- | --- | --- |
| `notice` | bool | show upgrade notice |
| `auto` | bool | auto-upgrade when allowed |
| `interval` | duration string | minimum interval between checks |
| `source` | string | `cdn` or `github` |

## Supported Segment Types

### SCM

`git`, `gitversion`, `jujutsu`, `mercurial`, `plastic`, `sapling`, `svn`, `fossil`

### Shell and system

`connection`, `executiontime`, `exit`, `os`, `path`, `project`, `root`, `session`, `shell`, `status`, `sysinfo`,
`text`, `time`, `upgrade`, `vim`, `winget`, `winreg`

### Cloud and infrastructure

`aws`, `az`, `azd`, `azfunc`, `cf`, `cftarget`, `gcp`, `helm`, `kubectl`, `pulumi`, `sitecore`, `talosctl`,
`terraform`

### Language and build tooling

`angular`, `argocd`, `aurelia`, `bazel`, `buf`, `bun`, `cds`, `clojure`, `cmake`, `copilot`, `crystal`, `dart`,
`deno`, `docker`, `dotnet`, `elixir`, `firebase`, `flutter`, `fortran`, `go`, `haskell`, `java`, `julia`,
`kotlin`, `lua`, `mojo`, `mvn`, `nbgv`, `nix-shell`, `nim`, `node`, `npm`, `nx`, `ocaml`, `perl`, `php`, `pnpm`,
`python`, `quasar`, `r`, `react`, `ruby`, `rust`, `svelte`, `swift`, `tauri`, `ui5tooling`, `umbraco`, `unity`,
`v`, `vala`, `xmake`, `yarn`, `zig`

### Web, APIs, and online services

`brewfather`, `carbonintensity`, `http`, `ipify`, `nba`, `nightscout`, `owm`, `spotify`, `strava`, `todoist`,
`wakatime`, `withings`, `ytm`, `lastfm`

## Parser Errors You Are Most Likely to Hit

- missing segment reference in `segments:`
- duplicate segment instance names
- unknown segment type
- missing `type` when inference fails
- invalid separator alias
- invalid mix of `style` with explicit separator fields

## Canonical Example

```yaml
cursor_padding: true
render_pending_icon: " "
render_pending_background: darkGray

daemon_timeout: 100
daemon_idle_timeout: "5"

prompt:
  - style: rounded
    segments: [session, path]

rprompt:
  - style: rounded
    segments: [git, time]

transient:
  - segments: [path.transient]

rtransient:
  - style: rounded
    segments: [git.transient, time.transient]

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
  background_templates:
    - "{{ if or (.Working.Changed) (.Staging.Changed) }}yellow{{ end }}"
  template: " {{ .HEAD }} "
  cache:
    duration: 30s
    strategy: folder
  options:
    fetch_status: true

path.transient:
  foreground: lightWhite
  background: transparent
  template: " {{ .Folder }} "

git.transient:
  foreground: lightWhite
  background: transparent
  template: " {{ .HEAD }} "
  options:
    fetch_status: true

time:
  foreground: white
  background: darkGray
  template: " {{ .LastDate | date \"15:04\" }} "

time.transient:
  foreground: darkGray
  background: transparent
  template: " {{ .LastDate | date \"15:04\" }} "
```
