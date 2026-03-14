# Configuration

## Canonical Format

`prompto` uses local YAML configuration files.

The main top-level layout keys are:

- `prompt`
- `rprompt`
- `secondary`
- `transient`
- `rtransient`

If you are coming from older oh-my-posh material, the main difference is that this fork documents the current layout
model directly instead of the older `blocks` model.

## Mental Model

A config has two layers:

1. Layout lines place named segments.
2. Top-level segment tables define how each named segment behaves.

```yaml
prompt:
  - segments: [session, path]

rprompt:
  - segments: [git, time]

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

time:
  foreground: white
  background: darkGray
  template: " {{ .LastDate | date \"15:04\" }} "
```

## Read This in Order

- [Quick start](./configuration/quick-start.md): build a practical config from scratch.
- [Layout](./configuration/layout.md): prompt lines, separators, and placement.
- [Segments](./configuration/segments.md): segment tables, naming, caching, timeouts, and pending rendering.
- [Templates](./configuration/templates.md): write text, conditionals, and cross-segment references.
- [Colors](./configuration/colors.md): color formats, palettes, color templates, and cycling.
- [Extras and shell features](./configuration/extras.md): transient prompts, tooltips, vim mode, title, daemon
  settings, and shell integration features.
- [Reference](./configuration/reference.md): exhaustive field reference and supported segment type list.
- [Segment reference](./segments/README.md): per-segment pages with type-specific options, properties, and examples.

## Minimal Working Example

```yaml
prompt:
  - segments: [path]

rprompt:
  - segments: [git]

transient:
  - segments: [path.transient]

path:
  foreground: white
  background: blue
  template: " {{ .Path }} "

git:
  foreground: black
  background_templates:
    - "{{ if or (.Working.Changed) (.Staging.Changed) }}yellow{{ else }}green{{ end }}"
  template: " {{ .HEAD }} "
  options:
    fetch_status: true

path.transient:
  foreground: lightWhite
  background: transparent
  template: " {{ .Folder }} "
```

`cursor_padding` is not shown here because the default is already `true`.

## Default Location

```text
macOS/Linux: ${XDG_CONFIG_HOME:-$HOME/.config}/prompto/config.yaml
Windows: %UserConfigDir%/prompto/config.yaml
```

## Practical Advice

- Start small and add segments one at a time.
- Give separate jobs separate segment names, for example `git` and `git.transient`.
- Use palettes once your config grows past a few segments.
- Keep your real config local and under version control.
- Use `prompto config image --output ./prompto-preview.png` when you want a quick visual preview.
