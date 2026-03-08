---
title: Windows Registry Key Query
description: Display the content of the requested Windows registry key.
---

## Segment Type

`winreg`

## What

Display the content of the requested Windows registry key.

Supported registry key types:

- `SZ` (displayed as string value)
- `EXPAND_SZ` (displayed as string value)
- `BINARY` (displayed as string value)
- `DWORD` (displayed in upper-case 0x hex)
- `QWORD` (displayed in upper-case 0x hex)

## Sample Configuration

```yaml
prompt:
  - segments: ["winreg"]

winreg:
  type: "winreg"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#444444"
  template: "  {{ .Value }}"
  options:
    path: "HKLM\\software\\microsoft\\windows nt\\currentversion\\buildlab"
    fallback: "unknown"
```

## Options

- `path`
  - Type: `string`
  - Description: registry path to the desired key using backslashes and with a valid root HKEY name. Ending path with \
    will get the (Default) key from that path
- `fallback`
  - Type: `string`
  - Description: the value to fall back to if no entry is found

## Template

### Default Template

```template
 {{ .Value }}
```

### Properties

- .Value
  - Type: `string`
  - Description: The result of your query, or fallback if not found.
