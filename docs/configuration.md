---
title: Configuration
description: Overview of the current prompto YAML configuration model, with links to detailed guides.
---

## Canonical Format

The current configuration model for this fork is:

- YAML only
- local file based
- layout driven through `prompt`, `rprompt`, `secondary`, `transient`, and `rtransient`

If you are familiar with the older website docs, the main difference is that this fork documents the current layout
parser rather than the older `blocks` model.

## Mental Model

A config has two layers:

1. Prompt layout lines place named segments.
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
- [Templates](./configuration/templates.md): global fields, helper functions, and cross-segment references.
- [Colors](./configuration/colors.md): color formats, palettes, color templates, and cycling.
- [Extras and shell features](./configuration/extras.md): transient prompts, tooltips, vim mode, title, upgrade,
  daemon settings, and shell-specific behavior.
- [Reference](./configuration/reference.md): exhaustive field reference and supported segment type list.
- [Segment reference](./segments/README.md): per-segment pages with type-specific options, properties, and examples.

## Minimal Working Example

```yaml
cursor_padding: true

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

## Default Location

```text
macOS/Linux: ${XDG_CONFIG_HOME:-$HOME/.config}/prompto/config.yaml
Windows: %UserConfigDir%/prompto/config.yaml
```

## Practical Advice

- Keep your main config local and under version control.
- Use one named segment per semantic job, then reuse that name in different prompt lines.
- Start with explicit segment names such as `git`, `git.transient`, and `time.rprompt`.
- Use `prompto config export --format yaml` when you want a normalized snapshot.
