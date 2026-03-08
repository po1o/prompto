---
title: Subversion
description: Display Subversion information when in a subversion repository. Also works for subfolders. For maximum compatibility, make sure your `svn` executable is up-to-date (when branch or status information is incorrect for example).
---

## Segment Type

`svn`

## What

Display [Subversion][svn] information when in a subversion repository. Also works for subfolders. For maximum
compatibility, make sure your `svn` executable is up-to-date (when branch or status information is incorrect for
example).

## Sample Configuration

```yaml
prompt:
  - segments: ["svn"]

svn:
  type: "svn"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#193549"
  background: "#ffeb3b"
  options:
    fetch_status: true
```

## Options

### Fetching information

As doing multiple [subversion][svn] calls can slow down the prompt experience, we do not fetch information by default.
You can set the following options to `true` to enable fetching additional information (and populate the template).

- `fetch_status`
  - Type: `boolean`
  - Default: `false`
  - Description: fetch the local changes
- `native_fallback`
  - Type: `boolean`
  - Default: `false`
  - Description: when set to `true` and `svn.exe` is not available when inside a WSL2 shared Windows drive, we will
    fallback to the native `svn` executable to fetch data. Not all information can be displayed in this case
- `status_formats`
  - Type: `map[string]string`
  - Description: a key, value map allowing to override how individual status items are displayed. For example,
    `"status_formats": { "Added": "Added: %d" }` will display the added count as `Added: 1` instead of `+1`. See the
    [Status](#status) section for available overrides

### Info

The fields `Repo`, `Branch` and `BaseRev` will still work with `fetch_status` set to `false`.

## Template

### Default Template

```template
 \ue0a0{{.Branch}} r{{.BaseRev}} {{.Working.String}}
```

### Properties

- `.Working`
  - Type: `Status`
  - Description: changes in the worktree (see below)
- `.Branch`
  - Type: `string`
  - Description: current branch (relative URL reported by `svn info`)
- `.BaseRev`
  - Type: `int`
  - Description: the currently checked out revision number
- `.Repo`
  - Type: `string`
  - Description: current repository (repos root URL reported by `svn info`)

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
- `.Moved`
  - Type: `int`
  - Description: number of changed moved files
- `.Conflicted`
  - Type: `int`
  - Description: number of changed tracked files with conflicts
- `.Changed`
  - Type: `boolean`
  - Description: if the status contains changes or not
- `.HasConflicts`
  - Type: `boolean`
  - Description: if the status contains conflicts or not
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
- `>`
  - Description: Moved
- `!`
  - Description: Conflicted

[svn]: https://subversion.apache.org
