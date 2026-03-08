---
title: GCP Context
description: Display the currently active GCP project, region and account
---

## Segment Type

`gcp`

## What

Display the currently active [GCP][gcp] project, region and account

## Sample Configuration

```yaml
prompt:
  - segments: ["gcp"]

gcp:
  type: "gcp"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#47888d"
  template: " 󱇶 {{.Project}} :: {{.Account}} "
```

## Template

### Default Template

```template
{{ if .Error }}{{ .Error }}{{ else }}{{ .Project }}{{ end }}
```

### Properties

- `.Project`
  - Type: `string`
  - Description: the currently active project
- `.Account`
  - Type: `string`
  - Description: the currently active account
- `.Region`
  - Type: `string`
  - Description: default region for the active context
- `.ActiveConfig`
  - Type: `string`
  - Description: the active configuration name
- `.Error`
  - Type: `string`
  - Description: contains any error messages generated when trying to load the GCP config

[gcp]: https://cloud.google.com/
