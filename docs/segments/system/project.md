---
title: Project
description: Display the current version of your project defined in the package file.
---

## Segment Type

`project`

## What

Display the current version of your project defined in the package file.

Supports:

- Node.js project (`package.json`)
- Deno project (`deno.json`, `deno.jsonc`)
- JSR project (`jsr.json`, `jsr.jsonc`)
- Cargo project (`Cargo.toml`)
- Python project (`pyproject.toml`, supports metadata defined according to [PEP 621][pep621-standard] or
  [Poetry][poetry-standard])
- Mojo project (`mojoproject.toml`)
- PHP project (`composer.json`)
- Dart project (`pubspec.yaml`)
- Any nuspec based project (`*.nuspec`, first file match info is displayed)
- .NET project (`*.sln`, `*.slnf`, `*.slnx`, `*.csproj`, `*.vbproj` or `*.fsproj`, first file match info is displayed)
- Julia project (`JuliaProject.toml`, `Project.toml`)
- PowerShell project (`*.psd1`, first file match info is displayed)

## Sample Configuration

```yaml
prompt:
  - segments: ["project"]

project:
  type: "project"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#193549"
  background: "#ffeb3b"
  template: " {{ if .Error }}{{ .Error }}{{ else }}{{ if .Version }} {{.Version}}{{ end }} {{ if .Name }}{{ .Name }}{{ end }}{{ end }} "
```

## Options

- `always_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: always show the segment
- `_files`
  - Type: `array`
  - Default: `[]`
  - Description: override the project's files to validate for. Use the `.Type` values listed below to override (e.g.
    `dotnet_files`)

## Template

### Default Template

```template
 {{ if .Error }}{{ .Error }}{{ else }}{{ if .Version }}\uf487 {{.Version}}{{ end }} {{ if .Name }}{{ .Name }}{{ end }}{{ end }}
```

### Properties

- `.Type`
  - Type: `string`
  - Description: The type of project:`node`; `deno`; `jsr`; `cargo`; `python`; `mojo`; `php`; `dart`; `nuspec`;
    `dotnet`; `julia`; `powershell`
- `.Version`
  - Type: `string`
  - Description: The version of your project
- `.Target`
  - Type: `string`
  - Description: The target framework/language version of your project
- `.Name`
  - Type: `string`
  - Description: The name of your project
- `.Error`
  - Type: `string`
  - Description: The error context when we can't fetch the project info

[pep621-standard]: https://peps.python.org/pep-0621/
[poetry-standard]: https://python-poetry.org/docs/pyproject/
