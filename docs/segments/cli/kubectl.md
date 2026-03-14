# Kubernetes

## Segment Type

`kubectl`

## What

Display the currently active [Kubernetes][kubernetes] context name and namespace name.

## Sample Configuration

```yaml
prompt:
  - segments: ["kubectl"]

kubectl:
  type: "kubectl"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#000000"
  background: "#ebcc34"
  template: " 󱃾 {{.Context}}{{if .Namespace}} :: {{.Namespace}}{{end}} "
  options:
    context_aliases:
      "arn:aws:eks:eu-west-1:1234567890:cluster/prompto": "prompto"
    cluster_aliases:
      "arn:aws:eks:eu-west-1:1234567890:cluster/prompto": "prompto-cluster"
```

## Options

- `display_error`
  - Type: `boolean`
  - Default: `false`
  - Description: show the error context when failing to retrieve the kubectl information
- `parse_kubeconfig`
  - Type: `boolean`
  - Default: `true`
  - Description: parse kubeconfig files instead of calling out to kubectl to improve performance
- `context_aliases`
  - Type: `object`
  - Description: map raw context names to the display names you want in the prompt
- `cluster_aliases`
  - Type: `object`
  - Description: map raw cluster names to the display names you want in the prompt

## Template

### Default Template

```template
{{ .Context }}{{ if .Namespace }} :: {{ .Namespace }}{{ end }}
```

### Properties

- `.Context`
  - Type: `string`
  - Description: the current kubectl context
- `.Namespace`
  - Type: `string`
  - Description: the current kubectl context namespace
- `.User`
  - Type: `string`
  - Description: the current kubectl context user
- `.Cluster`
  - Type: `string`
  - Description: the current kubectl context cluster

### Tip

It is common for the Kubernetes "default" namespace to be used when no namespace is provided. If you want your prompt to
render an empty current namespace using the word "default", you can use something like this for the template:

```text
{{.Context}} :: {{if .Namespace}}{{.Namespace}}{{else}}default{{end}}
```

[kubernetes]: https://kubernetes.io/
