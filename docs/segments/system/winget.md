# WinGet

## Segment Type

`winget`

## What

Displays the number of available [WinGet][winget] package updates. This segment only appears when there are updates
available.

### Info

This segment is only available on Windows.

## Sample Configuration

```yaml
prompt:
  - segments: ["winget"]

winget:
  type: "winget"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#0077c2"
  template: "  {{ .UpdateCount }} "
  cache:
    duration: "24h"
    strategy: "device"
```

## Template

### Default Template

```template
 \uf409 {{ .UpdateCount }}
```

### Properties

- `.UpdateCount`
  - Type: `int`
  - Description: the number of packages with available updates
- `.Updates`
  - Type: `[]WinGetPackage`
  - Description: array of packages with available updates

### WinGetPackage

- `.Name`
  - Type: `string`
  - Description: the package name
- `.ID`
  - Type: `string`
  - Description: the package ID
- `.Current`
  - Type: `string`
  - Description: the currently installed version
- `.Available`
  - Type: `string`
  - Description: the available version for update

[winget]: https://learn.microsoft.com/windows/package-manager/winget/
