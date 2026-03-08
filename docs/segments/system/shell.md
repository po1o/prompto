---
title: Shell
description: Show the current shell name (zsh, PowerShell, bash, ...).
---

## Segment Type

`shell`

## What

Show the current shell name (zsh, PowerShell, bash, ...).

## Sample Configuration

```yaml
prompt:
  - segments: ["shell"]

shell:
  type: "shell"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#0077c2"
  options:
    mapped_shell_names:
      pwsh: "PS"
```

## Options

- `mapped_shell_names`
  - Type: `object`
  - Description: custom glyph/text to use in place of specified shell names (case-insensitive)

## Template

### Default Template

```template
{{ .Name }}
```

### Properties

- `.Name`
  - Type: `string`
  - Description: the shell name
- `.Version`
  - Type: `string`
  - Description: the shell version
