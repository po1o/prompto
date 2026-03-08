---
title: Nerdbank.GitVersioning
description: Display the Nerdbank.GitVersioning version.
---

## Segment Type

`nbgv`

## What

Display the [Nerdbank.GitVersioning][nbgv] version.

### Caution

The Nerdbank.GitVersioning CLI can be a bit slow causing the prompt to feel slow.

## Sample Configuration

```yaml
prompt:
  - segments: ["nbgv"]

nbgv:
  type: "nbgv"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#3a579a"
  template: "  {{ .Version }} "
```

## Template

### Default Template

```template
 {{ .Version }}
```

### Properties

- `.Version`
  - Type: `string`
  - Description: the current version
- `.AssemblyVersion`
  - Type: `string`
  - Description: the current assembly version
- `.AssemblyInformationalVersion`
  - Type: `string`
  - Description: the current assembly informational version
- `.NuGetPackageVersion`
  - Type: `string`
  - Description: the current nuget package version
- `.ChocolateyPackageVersion`
  - Type: `string`
  - Description: the current chocolatey package version
- `.NpmPackageVersion`
  - Type: `string`
  - Description: the current npm package version
- `.SimpleVersion`
  - Type: `string`
  - Description: the current simple version

[nbgv]: https://github.com/dotnet/Nerdbank.GitVersioning
