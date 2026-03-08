---
title: Nix Shell
description: Displays the [nix shell] status if inside a nix-shell environment.
---

## Segment Type

`nix-shell`

## What

Displays the [nix shell] status if inside a nix-shell environment.

## Sample Configuration

```yaml
prompt:
  - segments: ["nix-shell"]

nix-shell:
  type: "nix-shell"
  style: "powerline"
  foreground: "blue"
  background: "transparent"
  template: "(󱄅-{{ .Type }})"
```

## Template

### Default Template

```template
via {{ .Type }}-shell"
```

### Properties

- `.Type`
  - Type: `string`
  - Description: the type of nix shell, can be `pure`, `impure` or `unknown`

[nix shell]: https://nixos.org/guides/nix-pills/developing-with-nix-shell.html
