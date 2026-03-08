---
title: Status Code
description: Displays the last known status code and/or the reason that the last command failed.
---

## Segment Type

`status`

## What

Displays the last known status code and/or the reason that the last command failed.

## Sample Configuration

```yaml
prompt:
  - segments: ["status"]

status:
  type: "status"
  style: "diamond"
  foreground: "#ffffff"
  background: "#00897b"
  background_templates: ["{{ if .Error }}#e91e63{{ end }}"]
  trailing_diamond: ""
  template: "<#193549></>  "
  options:
    always_enabled: true
```

## Options

- `always_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: always show the status
- `status_template`
  - Type: `string`
  - Default: `{{ .Code }}`
  - Description: [template][status-template] used to render an individual status code
- `status_separator`
  - Type: `string`
  - Default: `\|`
  - Description: used to separate multiple statuses when `$PIPESTATUS` is available

## Template

### Default Template

```template
 {{ .String }}
```

### Properties

- `.Code`
  - Type: `number`
  - Description: the last known exit code (command or pipestatus)
- `.String`
  - Type: `string`
  - Description: the formatted status codes using `status_template` and `status_separator`
- `.Error`
  - Type: `boolean`
  - Description: true if one of the commands has an error (validates on command status and pipestatus)

### Status Template

When using `status_template`, use `if eq .Code 0` to check for a successful exit code. The `.Error` property is used on
a global context and will not necessarily indicate that the current validated code is a non-zero value.

```template
{{ if eq .Code 0 }}\uf00c{{ else }}\uf071{{ end }}
```

In case you want the reason for the exit code instead of code itself, you can use the `reason` function:

```template
{{ if eq .Code 0 }}\uf00c{{ else }}\uf071 {{ reason .Code }}{{ end }}
```

[status-template]: #status-template
