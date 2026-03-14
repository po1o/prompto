# Pulumi

## Segment Type

`pulumi`

## What

Display the currently active [Pulumi][pulumi] logged-in user, url and stack.

### Caution

This requires a pulumi binary in your PATH and will only show in directories that contain a `Pulumi.yaml` file.

## Sample Configuration

```yaml
prompt:
  - segments: ["pulumi"]

pulumi:
  type: "pulumi"
  style: "diamond"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#662d91"
  template: " {{ .Stack }}{{if .User }} :: {{ .User }}@{{ end }}{{ if .URL }}{{ .URL }}{{ end }}"
```

## Options

- `fetch_stack`
  - Type: `boolean`
  - Default: `false`
  - Description: fetch the current stack name
- `fetch_about`
  - Type: `boolean`
  - Default: `false`
  - Description: fetch the URL and user for the current stask. Requires `fetch_stack` set to `true`

## Template

### Default Template

```template
\ue873 {{ .Stack }}{{if .User }} :: {{ .User }}@{{ end }}{{ if .URL }}{{ .URL }}{{ end }}
```

### Properties

- `.Stack`
  - Type: `string`
  - Description: the current stack name
- `.User`
  - Type: `string`
  - Description: is the current logged in user
- `.Url`
  - Type: `string`
  - Description: the URL of the state where pulumi stores resources

[pulumi]: https://www.pulumi.com/
