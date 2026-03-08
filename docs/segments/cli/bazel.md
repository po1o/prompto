---
title: Bazel
description: Display the currently active Bazel version.
---

## Segment Type

`bazel`

## What

Display the currently active [Bazel][bazel-github] version.

## Sample Configuration

```yaml
prompt:
  - segments: ["bazel"]

bazel:
  type: "bazel"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#43a047"
```

## Options

- `home_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: display the segment in the HOME folder or not
- `fetch_version`
  - Type: `boolean`
  - Default: `true`
  - Description: display the Bazel version - defaults to
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
  - Description: a go [text/template][go-text-template] [template][templates] that creates the URL of the version info
    documentation
- `icon`
  - Type: `string`
  - Default: `\ue63a`
  - Description: the icon for the segment
- `extensions`
  - Type: `[]string`
  - Default: `*.bazel, *.bzl, BUILD, WORKSPACE, .bazelrc, .bazelversion`
  - Description: allows to override the default list of file extensions to validate
- `folders`
  - Type: `[]string`
  - Default: `bazel-bin, bazel-out, bazel-testlogs`
  - Description: allows to override the list of folder names to validate
- `tooling`
  - Type: `[]string`
  - Default: `bazel`
  - Description: the tooling to use for fetching the version

## Template

### Default Template

```template
{{ if .Error }}{{ .Icon }} {{ .Error }}{{ else }}{{ url .Icon .URL }} {{ .Full }}{{ end }}
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
- `.Icon`
  - Type: `string`
  - Description: the icon representing Bazel's logo

[bazel-github]: https://github.com/bazelbuild/bazel
[go-text-template]: https://golang.org/pkg/text/template/
[templates]: ../../configuration/templates.md
[time.ParseDuration]: https://golang.org/pkg/time/#ParseDuration
