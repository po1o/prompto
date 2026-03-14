# AWS Context

## Segment Type

`aws`

## What

Display the currently active [AWS][aws] profile and region.

If only a region is known and `display_default` is left enabled, the segment falls back to the profile name `default`.

## Sample Configuration

```yaml
prompt:
  - segments: ["aws"]

aws:
  type: "aws"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#FFA400"
  template: "  {{.Profile}}{{if .Region}}@{{.Region}}{{end}}"
```

## Options

- `display_default`
  - Type: `boolean`
  - Default: `true`
  - Description: show the segment when the effective profile is `default`

## Template

### Default Template

```template
 {{ .Profile }}{{ if .Region }}@{{ .Region }}{{ end }}
```

### Properties

- `.Profile`
  - Type: `string`
  - Description: the currently active profile
- `.Region`
  - Type: `string`
  - Description: the currently active region
- `.RegionAlias`
  - Type: `string`
  - Description: short alias for the currently active region

[aws]: https://aws.amazon.com/
