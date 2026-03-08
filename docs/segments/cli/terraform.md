---
title: Terraform Context
description: Display the currently active Terraform Workspace name.
---

## Segment Type

`terraform`

## What

Display the currently active [Terraform][terraform] Workspace name.

## Sample Configuration

```yaml
prompt:
  - segments: ["terraform"]

terraform:
  type: "terraform"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#000000"
  background: "#ebcc34"
  template: "  {{.WorkspaceName}}"
```

## Options

- `fetch_version`
  - Type: `boolean`
  - Default: `false`
  - Description: fetch the version information from `versions.tf`, `main.tf` or `terraform.tfstate`
- `command`
  - Type: `string`
  - Default: `terraform`
  - Description: the command(s) to run, allows support for `tofu`

## Template

### Default Template

```template
 {{ .WorkspaceName }}{{ if .Version }} {{ .Version }}{{ end }}
```

### Properties

- `.WorkspaceName`
  - Type: `string`
  - Description: is the current workspace name
- `.Version`
  - Type: `string`
  - Description: terraform version (set `fetch_version` to `true`)

[terraform]: https://developer.hashicorp.com/terraform
