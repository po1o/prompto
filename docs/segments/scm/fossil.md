---
title: Fossil
description: Display Fossil information when in a fossil repository.
---

## Segment Type

`fossil`

## What

Display [Fossil][fossil] information when in a fossil repository.

## Sample Configuration

```yaml
prompt:
  - segments: ["fossil"]

fossil:
  type: "fossil"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#193549"
  background: "#ffeb3b"
```

## Options

- `native_fallback`
  - Type: `boolean`
  - Default: `false`
  - Description: when set to `true` and `fossil.exe` is not available when inside a WSL2 shared Windows drive, we will
    fallback to the native `fossil` executable to fetch data. Not all information can be displayed in this case

## Template

### Default Template

```template
 \ue725 {{.Branch}} {{.Status.String}}
```

### Properties

- `.Status`
  - Type: `FossilStatus`
  - Description: changes in the worktree (see below)
- `.Branch`
  - Type: `string`
  - Description: current branch

### FossilStatus

- `.Modified`
  - Type: `int`
  - Description: number of edited, updated and changed files
- `.Deleted`
  - Type: `int`
  - Description: number of deleted files
- `.Added`
  - Type: `int`
  - Description: number of added files
- `.Moved`
  - Type: `int`
  - Description: number of renamed files
- `.Conflicted`
  - Type: `int`
  - Description: number of conflicting files
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

- `+`
  - Description: added
- `!`
  - Description: conflicted
- `-`
  - Description: deleted
- `~`
  - Description: modified
- `>`
  - Description: moved

[fossil]: https://fossil-scm.org
