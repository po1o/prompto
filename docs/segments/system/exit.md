# Exit

## Segment Type

`exit`

## What

Display the last known exit code and/or the reason that the last command failed.

`exit` and `status` are equivalent segment types backed by the same writer. Use whichever name fits your config best.

## Sample Configuration

```yaml
prompt:
  - segments: [exit]

exit:
  type: exit
  style: diamond
  foreground: "#ffffff"
  background: "#00897b"
  background_templates:
    - "{{ if .Error }}#e91e63{{ end }}"
  trailing_diamond: ""
  template: " <#193549></>  "
  options:
    always_enabled: true
```

## Options

- `always_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: always show the segment, even when the exit code is zero
- `status_template`
  - Type: `string`
  - Default: `{{ .Code }}`
  - Description: template used to render one individual exit code
- `status_separator`
  - Type: `string`
  - Default: `|`
  - Description: separator used when pipe status values are available

## Template

### Default Template

```template
 {{ .String }}
```

### Properties

- `.Code`
  - Type: `number`
  - Description: the last known exit code for the command or pipeline element currently being rendered
- `.String`
  - Type: `string`
  - Description: the formatted status string built from `status_template` and `status_separator`
- `.Error`
  - Type: `boolean`
  - Description: true when one of the relevant exit codes is non-zero

### `status_template`

Use `status_template` when you want to customize how each code is rendered.
When you are inside `status_template`, test the current code with `if eq .Code 0`.

```template
{{ if eq .Code 0 }}{{ else }}{{ end }}
```

You can also render the shell reason instead of the raw number:

```template
{{ if eq .Code 0 }}{{ else }} {{ reason .Code }}{{ end }}
```

## Related Pages

- [Status Code](./status.md)
- [Templates](../../configuration/templates.md)
