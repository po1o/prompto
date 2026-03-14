# Templates

## What a Template Is

A template is the text recipe prompto uses to build a segment.
Plain text stays as-is.
Anything inside `{{ ... }}` is evaluated.

Templates are used in:

- `template`
- `templates`
- `foreground_templates`
- `background_templates`
- `console_title_template`
- `palettes.template`

## What Data You Can Use

Templates can read two kinds of data:

- global prompt data, such as the current folder, user, host, shell, and exit code
- segment-specific data, such as `.Path`, `.HEAD`, `.Profile`, or `.Mode`

If a segment and the global prompt data both define the same field name, the segment field wins.

## Global Fields

| Field | Meaning |
| --- | --- |
| `.Root` | whether the current user is root or admin |
| `.PWD` | current working directory with `~` shortening |
| `.AbsolutePWD` | current working directory without `~` shortening |
| `.PSWD` | PowerShell working directory when it is not a normal filesystem path |
| `.Folder` | current folder name |
| `.Shell` | current shell name after `maps.shell_name` rewriting |
| `.ShellVersion` | shell version |
| `.SHLVL` | shell nesting level |
| `.UserName` | current user name after `maps.user_name` rewriting |
| `.HostName` | host name after `maps.host_name` rewriting |
| `.Code` | last exit code |
| `.Jobs` | current background job count when the shell exposes it |
| `.OS` | operating system or Linux platform string |
| `.WSL` | whether the shell is running inside WSL |
| `.PromptCount` | prompt invocation count for this session |
| `.Version` | prompto version |
| `.Var` | values from the top-level `var:` map |
| `.Segments` | data from previously rendered segments |
| `.Segment` | metadata about the segment currently being rendered |

## Current Segment Metadata

| Field | Meaning |
| --- | --- |
| `.Segment.Index` | render order of the current segment |
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

## Basic Examples

### Conditional text

```yaml
status:
  template: " {{ if gt .Code 0 }}failed{{ else }}ok{{ end }} "
```

### Transforming text

```yaml
path:
  template: " {{ upper .Folder }} "
```

### Reusing the selected `time_format`

```yaml
time:
  template: " {{ .LastDate | date .Format }} "
  options:
    time_format: "15:04"
```

## Template Lists

`templates` lets you provide more than one template.
`templates_logic` decides how prompto uses the results.

### `join`

Render every non-empty result and concatenate them.

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

Use the first non-empty result and stop there.

```yaml
text.mode:
  type: text
  templates_logic: first_match
  templates:
    - "{{ if .Root }} ROOT {{ end }}"
    - "{{ if .WSL }} WSL {{ end }}"
    - " SHELL "
```

`foreground_templates` and `background_templates` also use first-match behavior.

## Common Helper Functions

prompto includes many helper functions.
These are the ones you are most likely to use directly:

| Function | Purpose |
| --- | --- |
| `date` | format a date or time |
| `dateInZone` | format a date or time in a specific time zone |
| `secondsRound` | turn seconds into a rounded duration string |
| `url` | create a terminal hyperlink |
| `path` | create a file hyperlink |
| `glob` | test text against a glob pattern |
| `matchP` | test text against a regular expression |
| `findP` | extract text with a regular expression |
| `replaceP` | replace text with a regular expression |
| `random` | choose one value from a list |
| `reason` | render a status reason from an exit code |
| `hresult` | convert a status code to HRESULT form |
| `trunc`, `truncE` | shorten text |
| `readFile` | read a file as text |
| `stat` | inspect file metadata |
| `dir`, `base` | extract parts of a filesystem path |

Example:

```yaml
executiontime:
  template: " {{ secondsRound .Ms }} "
```

## Reading `|` In a Template

The `|` character passes the value on the left into the expression on the right.

For example:

```template
{{ .LastDate | date .Format }}
```

reads as:

- take `.LastDate`
- format it with `date`
- use `.Format` as the chosen time format

## Cross-Segment References

You can read data from previously rendered segments through `.Segments`.
This is useful when one segment depends on another.

```yaml
status:
  template: " {{ if .Segments.Git.UpstreamGone }}gone{{ else if gt .Code 0 }}failed{{ else }}ok{{ end }} "
```

Keep these constraints in mind:

- the referenced segment must exist in the config
- the reference name must match the segment's runtime name
- cross-segment references create dependencies between segments

When you use two instances of the same segment type, give them stable names:

```yaml
git:
  template: " {{ .HEAD }} "

git.short:
  template: " {{ .HEAD }} "
```

Then reference exactly the one you mean.

## Where Else Templates Are Used

Templates are not limited to segment text.
They also power fields such as:

- `console_title_template`
- `palettes.template`
- `foreground_templates`
- `background_templates`

## Practical Advice

- Keep templates short.
- Move repeated values into `var`.
- Use `templates_logic: first_match` for fallback behavior.
- Use cross-segment references only when the dependency is clear and necessary.
- Prefer readable templates over dense one-liners.
