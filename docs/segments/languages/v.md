# V

## Segment Type

`v`

## What

Display the currently active [V][v-lang] version.

## Sample Configuration

```yaml
prompt:
  - segments: ["v"]

v:
  type: "v"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#193549"
  background: "#4F87FF"
  template: "  {{ .Full }} "
```

## Options

- `home_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: display the segment in the HOME folder or not
- `fetch_version`
  - Type: `boolean`
  - Default: `true`
  - Description: fetch the V version (`v --version`)
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
  - Default: `*.v`
  - Description: allows to override the default list of file extensions to validate
- `folders`
  - Type: `[]string`
  - Description: allows to override the list of folder names to validate
- `tooling`
  - Type: `[]string`
  - Default: `v`
  - Description: the tooling to use for fetching the version

## Template

### Default Template

```template
{{ if .Error }}{{ .Error }}{{ else }}{{ .Full }}{{ end }}
```

### Properties

- `.Full`
  - Type: `string`
  - Description: the full version (e.g., "0.4.9")
- `.Major`
  - Type: `string`
  - Description: major number (e.g., "0")
- `.Minor`
  - Type: `string`
  - Description: minor number (e.g., "4")
- `.Patch`
  - Type: `string`
  - Description: patch number (e.g., "9")
- `.Commit`
  - Type: `string`
  - Description: commit hash (e.g., "b487986")
- `.Error`
  - Type: `string`
  - Description: error encountered when fetching the version string

[v-lang]: https://vlang.io/
