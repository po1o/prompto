---
title: Azure Developer CLI
description: Display the currently active environment in the Azure Developer CLI.
---

## Segment Type

`azd`

## What

Display the currently active environment in the [Azure Developer CLI][azd].

## Sample Configuration

```yaml
prompt:
  - segments: ["azd"]

azd:
  type: "azd"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#000000"
  background: "#9ec3f0"
  template: "  {{ .DefaultEnvironment }} "
```

## Template

### Default Template

```template
 \uebd8 {{ .DefaultEnvironment }}
```

### Properties

- `.DefaultEnvironment`
  - Type: `string`
  - Description: Azure Developer CLI environment name
- `.Version`
  - Type: `number`
  - Description: Config version number

[azd]: https://aka.ms/azd
