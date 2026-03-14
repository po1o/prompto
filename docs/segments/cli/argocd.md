# ArgoCD Context

## Segment Type

`argocd`

## What

Display the current [ArgoCD][argocd] context name, user and/or server.

## Sample Configuration

```yaml
prompt:
  - segments: ["argocd"]

argocd:
  type: "argocd"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#FFA400"
  template: "  {{ .Name }}:{{ .User }}@{{ .Server }} "
```

## Template

### Default Template

```template
{{ .Name }}
```

### Properties

- `.Name`
  - Type: `string`
  - Description: the current context name
- `.Server`
  - Type: `string`
  - Description: the server of the current context
- `.User`
  - Type: `string`
  - Description: the user of the current context

[argocd]: https://argo-cd.readthedocs.io/en/stable/
