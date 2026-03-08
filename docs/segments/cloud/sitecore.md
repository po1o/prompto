---
title: Sitecore
description: Display current [Sitecore] environment. Will not be active when sitecore.json and user.json don't exist.
---

## Segment Type

`sitecore`

## What

Display current [Sitecore] environment. Will not be active when sitecore.json and user.json don't exist.

## Sample Configuration

```yaml
prompt:
  - segments: ["sitecore"]

sitecore:
  type: "sitecore"
  style: "plain"
  foreground: "#000000"
  background: "#FFFFFF"
  template: "Env: {{ .EndpointName }}{{ if .CmHost }} CM: {{ .CmHost }}{{ end }}"
```

## Options

- `display_default`
  - Type: `boolean`
  - Default: `true`
  - Description: display the segment or not when the Sitecore environment name matches `default`

## Template

### Default Template

```template
{{ .EndpointName }} {{ if .CmHost }}({{ .CmHost }}){{ end }}
```

### Properties

- `EndpointName`
  - Type: `string`
  - Description: name of the current Sitecore environment
- `CmHost`
  - Type: `string`
  - Description: host of the current Sitecore environment

[Sitecore]: https://www.sitecore.com/
