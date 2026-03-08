---
title: Layout
description: Define prompt lines, right prompts, transient prompts, separators, and multi-line layouts.
---

## Prompt Families

The layout layer is defined by these top-level arrays:

| Key | Alignment | Purpose |
| --- | --- | --- |
| `prompt` | left | Primary prompt shown before command entry |
| `rprompt` | right | Right-aligned primary prompt |
| `secondary` | left | Continuation prompt for multi-line input |
| `transient` | left | Replaces the previous primary prompt after Enter |
| `rtransient` | right | Right-aligned transient prompt |

Each item in those arrays is a prompt line object.

## Prompt Line Object

A line object supports:

- `segments`: required list of segment names
- `filler`: optional repeated fill text between left and right prompt content
- `style`: separator shortcut alias
- `leading_style`, `trailing_style`: explicit separator aliases
- `leading_separator`, `trailing_separator`: explicit separator glyphs

Example:

```yaml
prompt:
  - style: rounded
    segments: [session, path]

rprompt:
  - leading_style: rounded
    trailing_style: rounded
    segments: [git, time]
```

## `style` Shortcut Semantics

`style` is alignment-aware:

- On left-aligned lines (`prompt`, `secondary`, `transient`), it sets the trailing separator.
- On right-aligned lines (`rprompt`, `rtransient`), it sets the leading separator.

That means this:

```yaml
prompt:
  - style: rounded
    segments: [path]
```

is shorthand for a left prompt line that uses the `rounded` trailing separator.

## Mutual Exclusion Rules

A line cannot mix the shortcut and explicit separator forms.
These combinations are invalid:

- `style` together with `leading_style`
- `style` together with `trailing_style`
- `style` together with `leading_separator`
- `style` together with `trailing_separator`
- `leading_style` together with `leading_separator`
- `trailing_style` together with `trailing_separator`

## Separator Aliases

The supported separator aliases are:

- `powerline`
- `powerline_thin`
- `rounded`
- `rounded_thin`
- `slant`
- `block`
- `flame`
- `pixel`
- `lego`

See the full glyph table in [Reference](./reference.md#separator-aliases).

## Explicit Separators

If you want custom glyphs instead of a named alias:

```yaml
prompt:
  - leading_separator: "["
    trailing_separator: "]"
    segments: [text.prompt]

text.prompt:
  type: text
  foreground: white
  background: blue
  template: " prompto "
```

## Multi-Line Layouts

A prompt family can contain multiple lines:

```yaml
prompt:
  - style: rounded
    segments: [session, path]
  - segments: [status, text.separator]

rprompt:
  - style: rounded
    segments: [git, time]
```

Each array element is rendered as its own line in that prompt family.

## Filler

`filler` is repeated to span the space between left and right content.
A common use is a divider line.

```yaml
prompt:
  - filler: "─"
    segments: [path]

rprompt:
  - segments: [git]
```

## Transient Layout

Use `transient` and `rtransient` when you want the previous prompt line to collapse after command execution.
A common pattern is to keep only the path and a short git indicator.

```yaml
transient:
  - segments: [path.transient]

rtransient:
  - segments: [git.transient]
```

The transient prompt can contain pending rendering just like the primary prompt.

## Secondary Layout

`secondary` is the continuation prompt used for multi-line command entry.
Keep it visually simple.

```yaml
secondary:
  - segments: [text.secondary]

text.secondary:
  type: text
  foreground: darkGray
  background: transparent
  template: " > "
```

## What a Layout Line References

`segments:` contains segment names, not inline segment objects.
This means the names must exist as top-level segment tables.

```yaml
prompt:
  - segments: [path, git.main]

path:
  template: " {{ .Path }} "

git.main:
  template: " {{ .HEAD }} "
```

Missing references fail config parsing.
