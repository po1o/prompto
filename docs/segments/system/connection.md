# Connection

## Segment Type

`connection`

## What

Show details about the currently connected network.

### Info

Currently only supports Windows.

## Sample Configuration

```yaml
prompt:
  - segments: ["connection"]

connection:
  type: "connection"
  style: "powerline"
  background: "#8822ee"
  foreground: "#222222"
  powerline_symbol: ""
```

## Options

- `type`
  - Type: `string`
  - Default: `wifi\|ethernet`
  - Description: the connection types to try, joined with `|`. The first successful match is shown. Supported values are
    `wifi`, `ethernet`, `bluetooth`, and `cellular`

## Template

### Default Template

```template
 {{ if eq .Type \"wifi\"}}\uf1eb{{ else if eq .Type \"ethernet\"}}\ueba9{{ end }}
```

### Properties

- `.Type`
  - Type: `string`
  - Description: the resolved connection type
- `.Name`
  - Type: `string`
  - Description: the name of the connection
