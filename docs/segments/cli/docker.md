---
title: Docker
description: Display the current Docker context. Will not be active when using the default context.
---

## Segment Type

`docker`

## What

Display the current [Docker][docker] context. Will not be active when using the default context.

## Sample Configuration

```yaml
prompt:
  - segments: ["docker"]

docker:
  type: "docker"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#000000"
  background: "#0B59E7"
  template: "  {{ .Context }} "
```

## Options

- `display_mode`
  - Type: `string`
  - Default: `context`
  - Description: `files`: the segment is only displayed when a file `extensions` listed is present; `context`: displays
    the segment when a Docker context active
- `fetch_context`
  - Type: `boolean`
  - Default: `true`
  - Description: also fetch the current active Docker context when in the `files` display mode
- `extensions`
  - Type: `[]string`
  - Default: `compose.yml, compose.yaml, docker-compose.yml, docker-compose.yaml, Dockerfile`
  - Description: allows to override the default list of file extensions to validate

## Template

### Default Template

```template
\uf308 {{ .Context }}
```

### Properties

- `.Context`
  - Type: `string`
  - Description: the current active context

[docker]: https://www.docker.com/
