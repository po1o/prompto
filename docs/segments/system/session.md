---
title: Session
description: Show the current user and host name.
---

## Segment Type

`session`

## What

Show the current user and host name.

## Sample Configuration

```yaml
prompt:
  - segments: ["session"]

session:
  type: "session"
  style: "diamond"
  foreground: "#ffffff"
  background: "#c386f1"
  leading_diamond: ""
  trailing_diamond: ""
  template: "{{ if .SSHSession }} {{ end }}{{ .UserName }}"
```

## Template

### Default Template

```template
 {{ if .SSHSession }}\ueba9 {{ end }}{{ .UserName }}@{{ .HostName }}
```

### Properties

- `.UserName`
  - Type: `string`
  - Description: the current user's name
- `.HostName`
  - Type: `string`
  - Description: the current computer's name
- `.SSHSession`
  - Type: `boolean`
  - Description: active SSH session or not
- `.Root`
  - Type: `boolean`
  - Description: are you a root/admin user or not
