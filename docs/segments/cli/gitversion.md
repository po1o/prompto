# GitVersion

## Segment Type

`gitversion`

## What

Display the [GitVersion][gitversion] version. We _strongly_ recommend using [GitVersion Portable][gitversion-portable]
for this.

### Caution

The GitVersion CLI can be a bit slow, causing the prompt to feel slow. This is why we cache the value for 30 minutes by
default.

## Sample Configuration

```yaml
prompt:
  - segments: ["gitversion"]

gitversion:
  type: "gitversion"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#3a579b"
  template: "  {{ .MajorMinorPatch }} "
```

## Template

### Default Template

```template
{{ .MajorMinorPatch }}
```

### Properties

You can leverage all variables from the [GitVersion][gitversion] CLI. Have a look at their [documentation][docs] for
more information.

[gitversion]: https://github.com/GitTools/GitVersion
[gitversion-portable]: http://chocolatey.org/packages/GitVersion.Portable
[docs]: https://gitversion.net/docs/reference/variables
