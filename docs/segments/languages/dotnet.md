# Dotnet

## Segment Type

`dotnet`

## What

Display the currently active [.NET SDK][net-sdk-docs] version.

## Sample Configuration

```yaml
prompt:
  - segments: ["dotnet"]

dotnet:
  type: "dotnet"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#000000"
  background: "#00ffff"
  template: "  {{ .Full }} "
```

## Options

- `home_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: display the segment in the HOME folder or not
- `fetch_version`
  - Type: `boolean`
  - Default: `true`
  - Description: fetch the active version or not; useful if all you need is an icon indicating `dotnet`
- `cache_duration`
  - Type: `string`
  - Default: `none`
  - Description: how long to cache the version. Use values like `30s`, `5m`, or `1h`. Use `none` to disable caching
- `missing_command_text`
  - Type: `string`
  - Description: text to display when the command is missing
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
  - Default: `*.cs, *.csx, *.vb, *.fs, *.fsx, *.sln, *.slnf, *.slnx, *.csproj, *.fsproj, *.vbproj, global.json`
  - Description: allows to override the default list of file extensions to validate
- `folders`
  - Type: `[]string`
  - Description: allows to override the list of folder names to validate
- `tooling`
  - Type: `[]string`
  - Default: `dotnet`
  - Description: the tooling to use for fetching the version
- `fetch_sdk_version`
  - Type: `boolean`
  - Default: `false`
  - Description: fetch the SDK version in `global.json` when present

## Template

### Default Template

```template
{{ if .Unsupported }}\uf071{{ else }}{{ .Full }}{{ end }}
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
- `.Prerelease`
  - Type: `string`
  - Description: prerelease info text
- `.BuildMetadata`
  - Type: `string`
  - Description: build metadata
- `.URL`
  - Type: `string`
  - Description: URL of the version info / release notes
- `.SDKVersion`
  - Type: `string`
  - Description: the SDK version in `global.json` when `fetch_sdk_version` is `true`
- `.Error`
  - Type: `string`
  - Description: error encountered when fetching the version string

[net-sdk-docs]: https://docs.microsoft.com/en-us/dotnet/core/tools
