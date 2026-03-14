# Jujutsu

## Segment Type

`jujutsu`

## What

Display [Jujutsu][jujutsu] information when in a Jujutsu repository.

## Sample Configuration

```yaml
prompt:
  - segments: ["jujutsu"]

jujutsu:
  type: "jujutsu"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#193549"
  background: "#ffeb3b"
  options:
    fetch_status: true
    ignore_working_copy: false
    fetch_ahead_counter: true
    ahead_icon: "⇡"
```

## Options

### Fetching information

As doing Jujutsu (jj) calls can slow down the prompt experience, we do not fetch information by default. Set
`status_formats` to `true` to enable fetching additional information (and populate the template).

- `change_id_min_len`
  - Type: `int`
  - Default: `0`
  - Description: `ChangeID` will be at least this many characters, even if a shorter one would be unique
- `fetch_status`
  - Type: `boolean`
  - Default: `false`
  - Description: fetch the local changes
- `ignore_working_copy`
  - Type: `boolean`
  - Default: `true`
  - Description: don't snapshot/update the working copy
- `fetch_ahead_counter`
  - Type: `boolean`
  - Default: `false`
  - Description: fetch a counter for number of changes between working copy and closest bookmark
- `ahead_icon`
  - Type: `string`
  - Default: `\u21e1`
  - Description: icon/character between bookmark and ahead counter
- `native_fallback`
  - Type: `boolean`
  - Default: `false`
  - Description: when set to `true` and `jj.exe` is not available when inside a WSL2 shared Windows drive, we will
    fallback to the native `jj` executable to fetch data. Not all information can be displayed in this case
- `status_formats`
  - Type: `map[string]string`
  - Description: a key, value map allowing to override how individual status items are displayed. For example,
    `"status_formats": { "Added": "Added: %d" }` will display the added count as `Added: 1` instead of `+1`. See the
    [Status](#status) section for available overrides

## Template

### Default Template

```template
 \uf1fa{{.ChangeID}}{{if .Working.Changed}} \uf044 {{ .Working.String }}{{ end }}
```

### Properties

- `.Working`
  - Type: `Status`
  - Description: changes in the working copy (see below)
- `.ChangeID`
  - Type: `string`
  - Description: The shortest unique prefix of the working copy change that's at least change_id_min_len long
- `.ClosestBookmarks`
  - Type: `string`
  - Description: Closest bookmark(s) on ancestors

### Status

- `.Modified`
  - Type: `int`
  - Description: number of modified files
- `.Deleted`
  - Type: `int`
  - Description: number of deleted files
- `.Added`
  - Type: `int`
  - Description: number of added files
- `.Moved`
  - Type: `int`
  - Description: number of renamed files
- `.Changed`
  - Type: `boolean`
  - Description: if the status contains changes or not
- `.String`
  - Type: `string`
  - Description: a string representation of the changes above

Local changes use the following syntax:

- `~`
  - Description: Modified
- `-`
  - Description: Deleted
- `+`
  - Description: Added
- `>`
  - Description: Moved

[jujutsu]: https://www.jj-vcs.dev/
