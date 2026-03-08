---
title: Mercurial
description: Display Mercurial information when in a Mercurial repository. For maximum compatibility, make sure your `hg` executable is up-to-date (when branch or status information is incorrect for example).
---

## Segment Type

`mercurial`

## What

Display [Mercurial][mercurial] information when in a Mercurial repository. For maximum compatibility, make sure your
`hg` executable is up-to-date (when branch or status information is incorrect for example).

## Sample Configuration

```yaml
prompt:
  - segments: ["mercurial"]

mercurial:
  type: "mercurial"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#193549"
  background: "#ffeb3b"
  options:
    fetch_status: true
    native_fallback: false
```

## Options

### Fetching information

As doing Mercurial (hg) calls can slow down the prompt experience, we do not fetch information by default. You can set
`fetch_status` to `true` to enable fetching additional information (and populate the template).

- `fetch_status`
  - Type: `boolean`
  - Default: `false`
  - Description: fetch the local changes
- `native_fallback`
  - Type: `boolean`
  - Default: `false`
  - Description: when set to `true` and `hg.exe` is not available when inside a WSL2 shared Windows drive, we will
    fallback to the native `hg` executable to fetch data. Not all information can be displayed in this case
- `status_formats`
  - Type: `map[string]string`
  - Description: a key, value map allowing to override how individual status items are displayed. For example,
    `"status_formats": { "Added": "Added: %d" }` will display the added count as `Added: 1` instead of `+1`. See the
    [Status](#status) section for available overrides

## Template

### Default Template

```template
hg {{.Branch}} {{if .LocalCommitNumber}}({{.LocalCommitNumber}}:{{.ChangeSetIDShort}}){{end}}{{range .Bookmarks }} \uf02e {{.}}{{end}}{{range .Tags}} \uf02b {{.}}{{end}}{{if .Working.Changed}} \uf044 {{ .Working.String }}{{ end }}
```

### Properties

- `.Working`
  - Type: `Status`
  - Description: changes in the worktree (see below)
- `.IsTip`
  - Type: `boolean`
  - Description: Current commit is the tip commit
- `.ChangeSetID`
  - Type: `string`
  - Description: The current local commit number
- `.ChangeSetIDShort`
  - Type: `string`
  - Description: The current local commit number
- `.Branch`
  - Type: `string`
  - Description: current branch
- `.Bookmarks`
  - Type: `[]string`
  - Description: the currently checked out revision number
- `.Tags`
  - Type: `[]string`
  - Description: the currently checked out revision number

### Status

- `.Untracked`
  - Type: `int`
  - Description: number of files not under version control
- `.Modified`
  - Type: `int`
  - Description: number of modified files
- `.Deleted`
  - Type: `int`
  - Description: number of deleted files
- `.Added`
  - Type: `int`
  - Description: number of added files
- `.Changed`
  - Type: `boolean`
  - Description: if the status contains changes or not
- `.String`
  - Type: `string`
  - Description: a string representation of the changes above

Local changes use the following syntax:

- `?`
  - Description: Untracked
- `~`
  - Description: Modified
- `-`
  - Description: Deleted
- `+`
  - Description: Added

[mercurial]: https://www.mercurial-scm.org
