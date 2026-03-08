---
title: Go
description: Display the currently active Golang version.
---

## Segment Type

`go`

## What

Display the currently active [Golang][golang] version.

## Sample Configuration

```yaml
prompt:
  - segments: ["go"]

go:
  type: "go"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#7FD5EA"
  template: "  {{ .Full }} "
```

## Options

- `home_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: display the segment in the HOME folder or not
- `fetch_version`
  - Type: `boolean`
  - Default: `true`
  - Description: fetch the golang version
- `cache_duration`
  - Type: `string`
  - Default: `none`
  - Description: the duration for which the version will be cached. The duration is a string in the format `1h2m3s` and
    is parsed using the [time.ParseDuration] function from the Go standard library. To disable the cache, use `none`
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
  - Description: a go [text/template][go-text-template] [template][templates] that creates the URL of the version info /
    release notes
- `parse_mod_file`
  - Type: `boolean`
  - Default: `false`
  - Description: parse the go.mod file instead of calling `go version`
- `parse_go_work_file`
  - Type: `boolean`
  - Default: `false`
  - Description: parse the go.work file instead of calling `go version`
- `extensions`
  - Type: `[]string`
  - Default: `*.go, go.mod, go.work, go.sum, go.work.sum`
  - Description: allows to override the default list of file extensions to validate
- `folders`
  - Type: `[]string`
  - Description: allows to override the list of folder names to validate
- `tooling`
  - Type: `[]string`
  - Default: `mod, go`
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

[go-text-template]: https://golang.org/pkg/text/template/
[templates]: ../../configuration/templates.md
[golang]: https://go.dev/
[time.ParseDuration]: https://golang.org/pkg/time/#ParseDuration
