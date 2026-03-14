# Helm

## Segment Type

`helm`

## What

Display the version of [Helm][helm]

## Sample Configuration

```yaml
prompt:
  - segments: ["helm"]

helm:
  background: "#a7cae1"
  foreground: "#100e23"
  powerline_symbol: ""
  template: "  {{ .Version }}"
  style: "powerline"
  type: "helm"
```

## Options

- `display_mode`
  - Type: `string`
  - Default: `always`
  - Description: `always`: the segment is always displayed; `files`: the segment is only displayed when a chart source
    file `Chart.yaml` (or `Chart.yml`) or helmfile `helmfile.yaml` (or `helmfile.yml`) is present

## Template

### Default Template

```template
 Helm {{ .Version }}
```

### Properties

- `.Version`
  - Type: `string`
  - Description: Helm cli version

[helm]: https://helm.sh/
