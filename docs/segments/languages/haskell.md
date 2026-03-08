---
title: Haskell
description: Display the currently active Glasgow Haskell Compiler (GHC) version.
---

## Segment Type

`haskell`

## What

Display the currently active Glasgow [Haskell][haskell] Compiler (GHC) version.

## Sample Configuration

```yaml
prompt:
  - segments: ["haskell"]

haskell:
  type: "haskell"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#906cff"
  background: "#100e23"
  template: "  {{ .Full }}"
```

## Options

- `home_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: display the segment in the HOME folder or not
- `fetch_version`
  - Type: `boolean`
  - Default: `true`
  - Description: fetch the GHC version
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
- `stack_ghc_mode`
  - Type: `string`
  - Default: `never`
  - Description: determines when to use `stack ghc` to retrieve the version information. Using `stack ghc` will decrease
    performance.`never`: never use `stack ghc`; `package`: only use `stack ghc` when `stack.yaml` is in the root of the
    ; `always`: always use `stack ghc`
- `extensions`
  - Type: `[]string`
  - Default: `*.hs, *.lhs, stack.yaml, package.yaml, *.cabal, cabal.project`
  - Description: allows to override the default list of file extensions to validate
- `folders`
  - Type: `[]string`
  - Description: allows to override the list of folder names to validate
- `tooling`
  - Type: `[]string`
  - Default: `ghc`
  - Description: the tooling to use for fetching the version. Available options: `ghc`, `stack`

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
- `.StackGhc`
  - Type: `boolean`
  - Description: `true` if `stack ghc` was used, otherwise `false`

[go-text-template]: https://golang.org/pkg/text/template/
[templates]: ../../configuration/templates.md
[haskell]: https://www.haskell.org/
[time.ParseDuration]: https://golang.org/pkg/time/#ParseDuration
