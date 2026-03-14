# Zig

## Segment Type

`zig`

## What

Display the currently active [Zig][zig] version.

## Sample Configuration

```yaml
prompt:
  - segments: ["zig"]

zig:
  type: "zig"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#342311"
  background: "#ffad55"
  template: "  {{ if .Error }}{{ .Error }}{{ else }}{{ .Full }}{{ end }} "
```

## Options

- `home_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: display the segment in the HOME folder or not
- `fetch_version`
  - Type: `boolean`
  - Default: `true`
  - Description: fetch the zig version (`zig version`)
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
  - Default: `*.zig, *.zon`
  - Description: allows to override the default list of file extensions to validate
- `folders`
  - Type: `[]string`
  - Description: allows to override the list of folder names to validate
- `tooling`
  - Type: `[]string`
  - Default: `zig`
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
- `.Prerelease`
  - Type: `string`
  - Description: prerelease identifier
- `.BuildMetadata`
  - Type: `string`
  - Description: build identifier
- `.URL`
  - Type: `string`
  - Description: URL of the version info / release notes
- `.InProjectDir`
  - Type: `bool`
  - Description: whether the working directory is within a Zig project
- `.Error`
  - Type: `string`
  - Description: error encountered when fetching the version string

[zig]: https://ziglang.org/
