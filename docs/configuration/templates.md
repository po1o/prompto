---
title: Templates
description: Use Go templates, global fields, helper functions, environment variables, and cross-segment data.
---

## Template Engine

`prompto` templates use Go `text/template` plus:

- [Sprig](https://masterminds.github.io/sprig/)
- custom helper functions provided by `prompto`

Templates are used in:

- `template`
- `templates`
- `foreground_templates`
- `background_templates`
- `console_title_template`
- palette selection templates

## Global Fields

These fields are available in template context globally.
If a segment has a field with the same name, the segment field wins.

| Field | Meaning |
| --- | --- |
| `.Root` | whether the current user is root or admin |
| `.PWD` | current working directory with `~` expansion |
| `.AbsolutePWD` | current working directory without `~` shortening |
| `.PSWD` | non-filesystem PowerShell working directory when present |
| `.Folder` | current folder name |
| `.Shell` | current shell name after `maps.shell_name` mapping |
| `.ShellVersion` | shell version |
| `.SHLVL` | shell nesting level |
| `.UserName` | current user name after `maps.user_name` mapping |
| `.HostName` | host name after `maps.host_name` mapping |
| `.Code` | last exit code |
| `.Jobs` | current background job count when the shell exposes it |
| `.OS` | operating system or Linux platform string |
| `.WSL` | whether the shell is running inside WSL |
| `.PromptCount` | prompt invocation count for this session |
| `.Version` | `prompto` version |
| `.Var` | values from the top-level `var:` map |
| `.Segments` | previously rendered segment data available for cross-segment references |
| `.Segment` | metadata for the current segment |

## Current Segment Metadata

| Field | Meaning |
| --- | --- |
| `.Segment.Index` | render index of the current segment |
| `.Segment.Text` | rendered text of the current segment |

## Environment Variables

Environment variables are available through `.Env.NAME`.

```yaml
text.env:
  type: text
  template: " {{ .Env.HOME }} "
```

## Config Variables

Use top-level `var` for your own reusable values.

```yaml
var:
  repo_root: ~/development
  brand: prompto

text.brand:
  type: text
  template: " {{ .Var.brand }} "
```

## Basic Template Examples

### Conditional text

```yaml
status:
  template: " {{ if gt .Code 0 }}failed{{ else }}ok{{ end }} "
```

### Local variables

```yaml
path:
  template: " {{ $name := .Folder }}{{ upper $name }} "
```

## Template Lists

`templates` is a list of templates.
The `templates_logic` field controls how they are combined.

### `join`

Join every non-empty result:

```yaml
text.summary:
  type: text
  templates_logic: join
  templates:
    - "{{ if .Root }} root{{ end }}"
    - " {{ .UserName }}"
    - " on {{ .HostName }} "
```

### `first_match`

Use the first non-empty result and stop:

```yaml
text.mode:
  type: text
  templates_logic: first_match
  templates:
    - "{{ if .Root }} ROOT {{ end }}"
    - "{{ if .WSL }} WSL {{ end }}"
    - " SHELL "
```

`foreground_templates` and `background_templates` always behave like first-match fallbacks.

## Custom Helper Functions

Common helper functions include:

| Function | Purpose |
| --- | --- |
| `secondsRound` | turn seconds into a rounded duration string |
| `url` | emit a terminal hyperlink |
| `path` | emit a file hyperlink |
| `glob` | boolean glob test |
| `matchP` | regex match test |
| `findP` | regex find helper |
| `replaceP` | regex replacement |
| `random` | choose a random value from a list |
| `reason` | render a status reason from a code |
| `hresult` | convert a status code to HRESULT form |
| `trunc`, `truncE` | truncate text |
| `readFile` | read a file as text |
| `stat` | inspect file metadata |
| `dir`, `base` | path helpers |

Example:

```yaml
executiontime:
  template: " {{ secondsRound .Ms }} "
```

## Cross-Segment References

You can use previously rendered segment data from `.Segments`.
This lets one segment depend on another.

```yaml
status:
  template: " {{ if .Segments.Git.UpstreamGone }}gone{{ else if gt .Code 0 }}failed{{ else }}ok{{ end }} "
```

Important constraints:

- the referenced segment must exist in the config
- segment references create dependencies between segments
- use stable names so the dependency graph stays clear

When you need two different git segments, use different names:

```yaml
git:
  template: " {{ .HEAD }} "

git.short:
  template: " {{ .HEAD }} "
```

Then reference exactly the one you want.

## Templates in Titles and Palettes

Templates are not limited to segment text.
They also power fields such as:

- `console_title_template`
- `palettes.template`
- color template arrays

## Practical Advice

- Keep templates short and move repeated values into `var`.
- Use `templates_logic: first_match` for fallback logic.
- Use cross-segment references only when the dependency is explicit and necessary.
- Prefer readable templates over dense one-liners.
