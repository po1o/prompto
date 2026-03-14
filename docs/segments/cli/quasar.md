# Quasar

## Segment Type

`quasar`

## What

Display the currently active [Quasar CLI][quasar-cli] version. Only rendered when the current or parent folder contains
a `quasar.config` or `quasar.config.js` file.

## Sample Configuration

```yaml
prompt:
  - segments: ["quasar"]

quasar:
  type: "quasar"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#00B4FF"
  template: "  {{.Full}}{{ if .HasVite }}  {{ .Vite.Version }}{{ end }} "
```

## Options

- `home_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: display the segment in the HOME folder or not
- `missing_command_text`
  - Type: `string`
  - Description: text to display when the command is missing
- `fetch_version`
  - Type: `boolean`
  - Default: `true`
  - Description: fetch the NPM version
- `cache_duration`
  - Type: `string`
  - Default: `none`
  - Description: how long to cache the version. Use values like `30s`, `5m`, or `1h`. Use `none` to disable caching
- `display_mode`
  - Type: `string`
  - Default: `context`
  - Description: `always`: the segment is always displayed; `files`: the segment is only displayed when file
    `extensions` listed are present; `context`: displays the segment when the environment or files is active
- `version_url_template`
  - Type: `string`
  - Description: a template that builds the URL of the version information or release notes
- `fetch_dependencies`
  - Type: `boolean`
  - Default: `false`
  - Description: fetch the version number of the `vite` and `@quasar/app-vite` dependencies if present
- `extensions`
  - Type: `[]string`
  - Default: `quasar.config, quasar.config.js`
  - Description: allows to override the default list of file extensions to validate
- `folders`
  - Type: `[]string`
  - Description: allows to override the list of folder names to validate
- `tooling`
  - Type: `[]string`
  - Default: `quasar`
  - Description: the tooling to use for fetching the version

## Template

### Default Template

```template
 \ue87f {{.Full}}{{ if .HasVite }} \ueb29 {{ .Vite.Version }}{{ end }}
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
- `.Vite`
  - Type: `Dependency`
  - Description: the `vite` dependency, if found
- `.AppVite`
  - Type: `Dependency`
  - Description: the `@quasar/app-vite` dependency, if found

#### Dependency

- `.Version`
  - Type: `string`
  - Description: the full version
- `.Dev`
  - Type: `boolean`
  - Description: development dependency or not

[quasar-cli]: https://quasar.dev/start/quasar-cli
