---
title: Fortran
description: Display the currently active [fortran] compiler version.
---

## Segment Type

`fortran`

## What

Display the currently active [fortran] compiler version.

### Compiler Support

This only works with the [gfortran] compiler.

## Sample Configuration

```yaml
prompt:
  - segments: ["fortran"]

fortran:
  type: "fortran"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#422251"
  template: " 󱈚 {{ .Full }} "
```

## Options

- `home_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: display the segment in the HOME folder or not
- `fetch_version`
  - Type: `boolean`
  - Default: `true`
  - Description: fetch the gfortran version
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
- `extensions`
  - Type: `[]string`
  - Default: `fpm.toml, *.f, *.for, *.fpp, *.f77, *.f90, *.f95, *.f03, *.f08` + uppercase equivalents (`*.F` etc...)
  - Description: allows to override the default list of file extensions to validate
- `folders`
  - Type: `[]string`
  - Description: allows to override the list of folder names to validate
- `tooling`
  - Type: `[]string`
  - Default: `gfortran`
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
- `.URL`
  - Type: `string`
  - Description: URL of the version info / release notes
- `.Error`
  - Type: `string`
  - Description: error encountered when fetching the version string

[go-text-template]: https://golang.org/pkg/text/template/
[templates]: ../../configuration/templates.md
[fortran]: https://fortran-lang.org/
[gfortran]: https://fortranwiki.org/fortran/show/GFortran
[time.ParseDuration]: https://golang.org/pkg/time/#ParseDuration
