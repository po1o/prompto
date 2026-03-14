# CDS (SAP CAP)

## Segment Type

`cds`

## What

Display the active [CDS CLI][sap-cap-cds] version.

## Sample Configuration

```yaml
prompt:
  - segments: ["cds"]

cds:
  background: "#a7cae1"
  foreground: "#100e23"
  powerline_symbol: ""
  template: "  cds {{ .Full }} "
  style: "powerline"
  type: "cds"
```

## Options

- `home_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: display the segment in the HOME folder or not
- `fetch_version`
  - Type: `boolean`
  - Default: `true`
  - Description: fetch the CDS version
- `cache_duration`
  - Type: `string`
  - Default: `none`
  - Description: how long to cache the version. Use values like `30s`, `5m`, or `1h`. Use `none` to disable caching
- `missing_command_text`
  - Type: `string`
  - Description: text to display when the cds command is missing
- `display_mode`
  - Type: `string`
  - Default: `context`
  - Description: `always`: the segment is always displayed; `files`: the segment is only displayed when file
    `extensions` listed are present; `context`: displays the segment when the environment or files is active
- `version_url_template`
  - Type: `string`
  - Description: a template that builds the URL of the version information or release notes
- `extensions`
  - Type: `[]string`
  - Default: `.cdsrc.json, .cdsrc-private.json, *.cds`
  - Description: allows to override the default list of file extensions to validate
- `folders`
  - Type: `[]string`
  - Description: allows to override the list of folder names to validate
- `tooling`
  - Type: `[]string`
  - Default: `cds`
  - Description: the tooling to use for fetching the version

## Template

### Default Template

```template
{{ if .Error }}{{ .Error }}{{ else }}{{ .Full }}{{ end }}
```

### Properties

- `.Full`
  - Type: `string`
  - Description: the full version
- `.Major`
  - Type: `string`
  - Description: major number
- `.Minor`
  - Type: `string`
  - Description: minor number
- `.Patch`
  - Type: `string`
  - Description: patch number
- `.Error`
  - Type: `string`
  - Description: error encountered when fetching the version string
- `.HasDependency`
  - Type: `bool`
  - Description: a flag if `@sap/cds` was found in `package.json`

[sap-cap-cds]: https://cap.cloud.sap/docs/tools/#command-line-interface-cli
