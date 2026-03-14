# Plastic SCM

## Segment Type

`plastic`

## What

Display [Plastic SCM][plastic-scm] information when in a plastic repository. Also works for subfolders. For maximum
compatibility, make sure your `cm` executable is up-to-date (when branch or status information is incorrect for
example).

## Sample Configuration

```yaml
prompt:
  - segments: ["plastic"]

plastic:
  type: "plastic"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#193549"
  background: "#ffeb3b"
  background_templates: ["{{ if .MergePending }}#006060{{ end }}", "{{ if .Changed }}#FF9248{{ end }}", "{{ if and .Changed .Behind }}#ff4500{{ end }}", "{{ if .Behind }}#B388FF{{ end }}"]
  template: "{{ .Selector }}{{ if .Status.Changed }}  {{ end }}{{ .Status.String }}"
  options:
    fetch_status: true
```

## Plastic SCM Icon

If you want to use the icon of Plastic SCM in the segment, then please help me push the icon in this [issue][fa-issue]
by leaving a like! ![icon](https://www.plasticscm.com/images/icon-logo-plasticscm.svg)

## Options

### Fetching information

As doing multiple `cm` calls can slow down the prompt experience, we do not fetch information by default. You can set
the following property to `true` to enable fetching additional information (and populate the template).

- `fetch_status`
  - Type: `boolean`
  - Default: `false`
  - Description: fetch the local changes
- `native_fallback`
  - Type: `boolean`
  - Default: `false`
  - Description: when set to `true` and `cm.exe` is not available when inside a WSL2 shared Windows drive, we will
    fallback to the native `cm` executable to fetch data. Not all information can be displayed in this case
- `status_formats`
  - Type: `map[string]string`
  - Description: a key, value map allowing to override how individual status items are displayed. For example,
    `"status_formats": { "Added": "Added: %d" }` will display the added count as `Added: 1` instead of `+1`. See the
    [Status](#status) section for available overrides

### Icons

#### Branch

- `branch_icon`
  - Type: `string`
  - Default: `\uE0A0`
  - Description: the icon to use in front of the git branch name
- `mapped_branches`
  - Type: `object`
  - Description: custom glyph/text for specific branches. You can use `*` at the end as a wildcard character for
    matching
- `branch_template`
  - Type: `string`
  - Description: a [template][templates] to format that branch name. You can use `{{ .Branch }}` as reference to the
    original branch name

#### Selector

- `commit_icon`
  - Type: `string`
  - Default: `\uF417`
  - Description: icon/text to display before the commit context (detached HEAD)
- `tag_icon`
  - Type: `string`
  - Default: `\uF412`
  - Description: icon/text to display before the tag context

## Template

### Default Template

```template
 {{ .Selector }}
```

### Properties

- `.Selector`
  - Type: `string`
  - Description: the current selector context (branch/changeset/label)
- `.Behind`
  - Type: `bool`
  - Description: the current workspace is behind and changes are incoming
- `.Status`
  - Type: `Status`
  - Description: changes in the workspace (see below)
- `.MergePending`
  - Type: `bool`
  - Description: if a merge is pending and needs to be committed (known issue: when no file is left after a
    _Change/Delete conflict_ merge, the `MergePending` property is not set)

### Status

- `.Unmerged`
  - Type: `int`
  - Description: number of unmerged changes
- `.Deleted`
  - Type: `int`
  - Description: number of deleted changes
- `.Added`
  - Type: `int`
  - Description: number of added changes
- `.Modified`
  - Type: `int`
  - Description: number of modified changes
- `.Moved`
  - Type: `int`
  - Description: number of moved changes
- `.Changed`
  - Type: `boolean`
  - Description: if the status contains changes or not
- `.String`
  - Type: `string`
  - Description: a string representation of the changes above

Local changes use the following syntax:

- `x`
  - Description: Unmerged
- `-`
  - Description: Deleted
- `+`
  - Description: Added
- `~`
  - Description: Modified
- `v`
  - Description: Moved

[templates]: ../../configuration/templates.md
[plastic-scm]: https://www.plasticscm.com/
[fa-issue]: https://github.com/FortAwesome/Font-Awesome/issues/18504
