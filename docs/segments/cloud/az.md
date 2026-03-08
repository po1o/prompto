---
title: Azure Subscription
description: Display the currently active Azure subscription information.
---

## Segment Type

`az`

## What

Display the currently active [Azure][azure] subscription information.

## Sample Configuration

```yaml
prompt:
  - segments: ["az"]

az:
  type: "az"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#000000"
  background: "#9ec3f0"
  template: "  {{ .EnvironmentName }}"
  options:
    source: "pwsh"
```

## Options

- `source`
  - Type: `string`
  - Default: `cli&#124;pwsh`
  - Description: sources to get subscription information from. Can be any of the following values, joined by `&#124;` to
    loop multiple sources for context. `cli`: fetch the information from the CLI config; `pwsh`: fetch the information
    from the PowerShell Module config

## Template

### Default Template

```template
{{ .Name }}
```

### Properties

- `.EnvironmentName`
  - Type: `string`
  - Description: Azure environment name
- `.HomeTenantID`
  - Type: `string`
  - Description: home tenant id
- `.ID`
  - Type: `string`
  - Description: subscription id
- `.IsDefault`
  - Type: `boolean`
  - Description: is the default subscription or not
- `.Name`
  - Type: `string`
  - Description: subscription name
- `.State`
  - Type: `string`
  - Description: subscription state
- `.TenantID`
  - Type: `string`
  - Description: tenant id
- `.TenantDisplayName`
  - Type: `string`
  - Description: tenant name
- `.User.Name`
  - Type: `string`
  - Description: user name
- `.User.Type`
  - Type: `string`
  - Description: user type
- `.Origin`
  - Type: `string`
  - Description: where we received the information from, can be `CLI` or `PWSH`

[azure]: https://azure.microsoft.com
