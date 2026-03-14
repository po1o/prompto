# Dart

## Segment Type

`dart`

## What

Display the currently active [Dart][dart] version. Supports [fvm].

## Sample Configuration

```yaml
prompt:
  - segments: ["dart"]

dart:
  type: "dart"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#06A4CE"
  template: "  {{ .Full }} "
```

## Options

- `home_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: display the segment in the HOME folder or not
- `fetch_version`
  - Type: `boolean`
  - Default: `true`
  - Description: fetch the dart version
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
  - Default: `*.dart, pubspec.yaml, pubspec.yml, pubspec.lock`
  - Description: allows to override the default list of file extensions to validate
- `folders`
  - Type: `[]string`
  - Default: `.dart_tool`
  - Description: allows to override the list of folder names to validate
- `tooling`
  - Type: `[]string`
  - Default: `fvm, dart`
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
- `.URL`
  - Type: `string`
  - Description: URL of the version info / release notes
- `.Error`
  - Type: `string`
  - Description: error encountered when fetching the version string

[dart]: https://dart.dev/
[fvm]: https://fvm.app/
