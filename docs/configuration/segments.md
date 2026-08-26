# Segments

## What a Segment Definition Is

A segment definition is a top-level YAML table that tells `prompto` how to render one segment instance.
Layout lines refer to these definitions by name.

For segment-type-specific options, properties, and examples, use the [segment reference](../segments/README.md).

## Two Equivalent Definition Styles

### Flat name

```yaml
git.transient:
  template: " {{ .HEAD }} "
```

### Nested by type

```yaml
git:
  transient:
    template: " {{ .HEAD }} "
```

Both create the segment named `git.transient`.

## Type Inference

`type` is optional when the parser can infer it from the table name.

These infer their type automatically:

- `path:` infers `type: path`
- `git:` infers `type: git`
- `git.transient:` infers `type: git`
- nested form under a known type, such as `git: transient:`

If inference fails, parsing fails.

## Use Explicit Names for Distinct Jobs

When the same segment type is used in multiple places, give each instance a meaningful name.

```yaml
prompt:
  - segments: [path, git]

rtransient:
  - segments: [git.short, time.short]

git:
  template: " {{ .HEAD }} {{ .Working.String }} "
  options:
    fetch_status: true

git.short:
  template: " {{ .HEAD }} "
  options:
    fetch_status: true

time.short:
  type: time
  template: " {{ .LastDate | date \"15:04\" }} "
```

## Common Segment Fields

### Text and rendering

- `type`
- `alias`
- `template`
- `templates`
- `templates_logic`
- `style`
- `leading_style`, `trailing_style`
- `leading_separator`, `trailing_separator`
- `foreground`, `background`
- `foreground_templates`, `background_templates`

### Behavior

- `options`
- `cache`
- `timeout`
- `interactive`
- `force`
- `toggled`
- `keep_when_empty`
- `include_folders`, `exclude_folders`
- `min_width`, `max_width`
- `render_pending_icon`, `render_pending_background`

The complete list is in [Reference](./reference.md#segment-fields).

## `template` vs `templates`

Use `template` for one final text template.
Use `templates` when you want a list of templates that are either joined together or resolved with first-match logic.

```yaml
text.summary:
  type: text
  templates_logic: join
  templates:
    - "{{ if .Root }} root{{ end }}"
    - " {{ .UserName }}"
    - " on {{ .HostName }} "
```

## `properties` vs `options`

Use `options`.
`properties` is still accepted on input for backward compatibility, but it is normalized internally.

## Aliases

`alias` lets you decouple the runtime identity from the table name.
This matters for toggles and cross-segment references.

```yaml
git.main:
  alias: repo
  template: " {{ .HEAD }} "
```

If `alias` is omitted, the segment name is used.

## Folder Filters

`include_folders` and `exclude_folders` are anchored regex patterns matched against the current working directory.
Important details:

- matching is against the full current path
- `~` expands to the home directory
- matching is case-insensitive on macOS and Windows

```yaml
git:
  include_folders:
    - "~/development/.*"
  exclude_folders:
    - "~/development/archive/.*"
  template: " {{ .HEAD }} "
```

## Width Filters

Hide a segment outside a terminal width range:

```yaml
time:
  min_width: 100
  template: " {{ .LastDate | date \"15:04\" }} "
```

Rules:

- `min_width`: hide when the terminal is narrower than this value
- `max_width`: hide when the terminal is wider than this value
- both together: only show inside that range

## Timeout

`timeout` is in milliseconds.
If a segment exceeds the timeout, `prompto` stops waiting and kills child processes started for that segment.

```yaml
git:
  timeout: 1500
  options:
    fetch_status: true
```

## Cache

Use `cache` for expensive segments.

```yaml
git:
  cache:
    duration: 30s
    strategy: folder
```

Available strategies:

- `folder`: cache per working context, using the segment's folder-scoped cache key
- `session`: cache once per shell session
- `device`: cache across sessions on the machine

In practice:

- `folder` is the safest default for path-sensitive segments such as `git`
- `session` is useful when the value only needs to stay stable during one shell session
- `device` is useful for slow, globally stable information

## Pending Rendering

Slow segments can be rendered in a pending state when prompt streaming is active.
The pending text is:

- cached text, if available
- otherwise `...`

And it is prefixed by a pending icon.

```yaml
render_pending_icon: " "
render_pending_background: darkGray

git:
  render_pending_icon: " "
  render_pending_background: yellow
```

Resolution order:

1. Segment-level pending setting
2. Global pending setting
3. Built-in default for the icon

## Toggled Segments

A segment with `toggled: true` starts disabled.
Users can toggle it back on at runtime.

```yaml
aws:
  toggled: true
  template: " {{ .Profile }} "
```

Toggle from the CLI:

```bash
prompto toggle aws
```

## `force`

Normally a segment with only whitespace resolves to disabled.
Set `force: true` when you want it to render anyway.

## Separators and the Ribbon

A Powerline-style prompt is a run of interlocking blocks. Whether two neighbours interlock depends on how
the *second* one opens:

| The next segment has | Its neighbour's trailing separator is drawn on |
| --- | --- |
| a trailing separator and no leading one | that segment's background — the blocks interlock |
| a leading separator | the terminal background — each block keeps its own outline |
| no separators at all | the terminal background |

A segment that opens with a leading separator is already drawing its own edge, so filling the separator
behind it would put a block of color behind that outline. A segment with no separators is bare text that
happens to have colors, not a link in the ribbon.

The practical rule: **for a continuous ribbon, give every segment a trailing separator and leave its
leading separator unset.** Set a leading separator only on a segment that should open against the
terminal — the first segment of a line, or one that starts a new run.

This also applies to a segment that draws its own opening inside its `template` rather than with
`leading_separator`. Several bundled themes do that with `<transparent>` markup; such a segment must leave
both separators unset, or the separator before it gains a filled block behind the outline the template
draws.

## `keep_when_empty`

A segment that renders no text disappears, and the segments around it close the gap.
Set `keep_when_empty: true` to draw its separators anyway, with no text between them.

```yaml
git:
  keep_when_empty: true
  leading_separator: "("
  trailing_separator: ")"
  template: " {{ .HEAD }} "
```

Outside a repository this renders `()` instead of vanishing, so the rest of the prompt keeps its position.
Use it when a segment appears and disappears often and the movement is distracting.

`force` and `keep_when_empty` solve different problems: `force` makes an empty segment render its text anyway,
`keep_when_empty` keeps the shape of a segment that has no text to render.

## `interactive`

`interactive: true` switches the segment into the terminal writer's interactive mode.
Use this only when you explicitly need that behavior for the rendered content.

## Reuse Across Prompt Types

You can define multiple instances of the same segment type for primary, right, transient, and tooltip use.
With shared providers, expensive segment types can reuse one computation inside the same render.

```yaml
prompt:
  - segments: [git]

rtransient:
  - segments: [git.transient]

git:
  template: " {{ .HEAD }} {{ .Working.String }} "
  options:
    fetch_status: true

git.transient:
  template: " {{ .HEAD }} "
  options:
    fetch_status: true
```
