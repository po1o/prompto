# Node

## Segment Type

`node`

## What

Display the currently active [Node.js][node-js] version.

## Sample Configuration

```yaml
prompt:
  - segments: ["node"]

node:
  type: "node"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#6CA35E"
  template: "  {{ .Full }} "
```

## Options

- `home_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: display the segment in the HOME folder or not
- `fetch_version`
  - Type: `boolean`
  - Default: `true`
  - Description: fetch the Node.js version
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
- `fetch_package_manager`
  - Type: `boolean`
  - Default: `false`
  - Description: define if the current project uses PNPM, Yarn, or NPM
- `pnpm_icon`
  - Type: `string`
  - Default: `\ue865`
  - Description: the icon/text to display when using PNPM
- `yarn_icon`
  - Type: `string`
  - Default: `\ue6a7`
  - Description: the icon/text to display when using Yarn
- `npm_icon`
  - Type: `string`
  - Default: `\uE71E`
  - Description: the icon/text to display when using NPM
- `bun_icon`
  - Type: `string`
  - Default: `\ue76f`
  - Description: the icon/text to display when using Bun
- `extensions`
  - Type: `[]string`
  - Default: `*.js, *.ts, package.json, .nvmrc, pnpm-workspace.yaml, .pnpmfile.cjs, .vue`
  - Description: allows to override the default list of file extensions to validate
- `folders`
  - Type: `[]string`
  - Description: allows to override the list of folder names to validate
- `tooling`
  - Type: `[]string`
  - Default: `node`
  - Description: the tooling to use for fetching the version

## Template

### Default Template

```template
{{ if .PackageManagerIcon }}{{ .PackageManagerIcon }} {{ end }}{{ .Full }}
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
- `.PackageManagerName`
  - Type: `string`
  - Description: the package manager name (`bun`, `npm`, `yarn` or `pnpm`) when setting `fetch_package_manager` to
    `true`
- `.PackageManagerIcon`
  - Type: `string`
  - Description: the PNPM, Yarn, Bun, or NPM icon when setting `fetch_package_manager` to `true`
- `.Mismatch`
  - Type: `boolean`
  - Description: true if the version in `.nvmrc` is not equal to `.Full`
- `.Expected`
  - Type: `string`
  - Description: the expected version set in `.nvmrc`

[node-js]: https://nodejs.org
