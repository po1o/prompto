---
title: Cloud Foundry Target
description: Display the details of the logged Cloud Foundry endpoint (`cf target` details).
---

## Segment Type

`cftarget`

## What

Display the details of the logged [Cloud Foundry endpoint][cf-target] (`cf target` details).

## Sample Configuration

```yaml
prompt:
  - segments: ["cftarget"]

cftarget:
  background: "#a7cae1"
  foreground: "#100e23"
  powerline_symbol: ""
  template: "  {{ .Org }}/{{ .Space }} "
  style: "powerline"
  type: "cftarget"
```

## Options

- `display_mode`
  - Type: `string`
  - Default: `always`
  - Description: `always`: the segment is always displayed; `files`: the segment is only displayed when a `manifest.yml`
    file is present (or defined otherwise using `files`)
- `files`
  - Type: `[]string`
  - Default: `["manifest.yml"]`
  - Description: on which files to display the segment on. Will look in parent folders as well

## Template

### Default Template

```template
{{if .Org }}{{ .Org }}{{ end }}{{ if .Space }}/{{ .Space }}{{ end }}
```

### Properties

- `.Org`
  - Type: `string`
  - Description: Cloud Foundry organization
- `.Space`
  - Type: `string`
  - Description: Cloud Foundry space
- `.URL`
  - Type: `string`
  - Description: Cloud Foundry API URL
- `.User`
  - Type: `string`
  - Description: logged in user

[cf-target]: https://cli.cloudfoundry.org/en-US/v8/target.html
