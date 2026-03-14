# Sapling

## Segment Type

`sapling`

## What

Display [Sapling][sapling] information when in a sapling repository.

## Sample Configuration

```yaml
prompt:
  - segments: ["sapling"]

sapling:
  type: "sapling"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#193549"
  background: "#4C9642"
  background_templates: ["{{ if .Bookmark }}#4C9642{{ end }}"]
  options:
    fetch_status: true
```

## Options

### Fetching information

- `fetch_status`
  - Type: `boolean`
  - Default: `true`
  - Description: fetch the local changes - defaults to
- `native_fallback`
  - Type: `boolean`
  - Default: `false`
  - Description: when set to `true` and `sl.exe` is not available when inside a WSL2 shared Windows drive, we will
    fallback to the native `sl` executable to fetch data. Not all information can be displayed in this case
- `status_formats`
  - Type: `map[string]string`
  - Description: a key, value map allowing to override how individual status items are displayed. For example,
    `"status_formats": { "Added": "Added: %d" }` will display the added count as `Added: 1` instead of `+1`. See the
    [Status](#status) section for available overrides

## Template

### Default Template

```template
 {{ if .Bookmark }}\uf097 {{ .Bookmark }}*{{ else }}\ue729 {{ .ShortHash }}{{ end }}{{ if .Working.Changed }} \uf044 {{ .Working.String }}{{ end }}
```

### Properties

- `.RepoName`
  - Type: `string`
  - Description: the repo folder name
- `.Working`
  - Type: `Status`
  - Description: changes in the worktree (see below)
- `.Description`
  - Type: `string`
  - Description: the first line of the commit's description
- `.Author`
  - Type: `string`
  - Description: the author of the commit
- `.Hash`
  - Type: `string`
  - Description: the full hash of the commit
- `.ShortHash`
  - Type: `string`
  - Description: the short hash of the commit
- `.When`
  - Type: `string`
  - Description: the commit's relative time indication
- `.Bookmark`
  - Type: `string`
  - Description: the commit's bookmark (if any)
- `.Dir`
  - Type: `string`
  - Description: the repository's root directory
- `.RelativeDir`
  - Type: `string`
  - Description: the current directory relative to the root directory
- `.New`
  - Type: `boolean`
  - Description: true when there are no commits in the repo

### Status

- `.Modified`
  - Type: `int`
  - Description: number of modified changes
- `.Added`
  - Type: `int`
  - Description: number of added changes
- `.Deleted`
  - Type: `int`
  - Description: number of removed changes
- `.Untracked`
  - Type: `boolean`
  - Description: number of untracked changes
- `.Clean`
  - Type: `int`
  - Description: number of clean changes
- `.Missing`
  - Type: `int`
  - Description: number of missing changes
- `.Ignored`
  - Type: `boolean`
  - Description: number of ignored changes
- `.String`
  - Type: `string`
  - Description: a string representation of the changes above

Local changes use the following syntax:

- `~`
  - Description: Modified
- `+`
  - Description: Added
- `-`
  - Description: Deleted
- `?`
  - Description: Untracked
- `=`
  - Description: Clean
- `!`
  - Description: Missing
- `Ø`
  - Description: Ignored

[sapling]: https://sapling-scm.com/
