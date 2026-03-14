# Ipify

## Segment Type

`ipify`

## What

[Ipify][ipify] is a simple Public IP Address API, it returns your public IP Address in plain text.

## Sample Configuration

```yaml
prompt:
  - segments: ["ipify"]

ipify:
  type: "ipify"
  style: "diamond"
  foreground: "#ffffff"
  background: "#c386f1"
  leading_diamond: ""
  trailing_diamond: ""
  template: "{{ .IP }}"
  options:
    http_timeout: 1000
```

## Options

- `url`
  - Type: `string`
  - Default: `https://api.ipify.org`
  - Description: The Ipify URL, by default IPv4 is used, use `https://api64.ipify.org` for IPv6
- `http_timeout`
  - Type: `int`
  - Default: `20`
  - Description: in milliseconds - how long may the segment wait for a response of the ipify API
- `cache_duration`
  - Type: `string`
  - Default: `24h`
  - Description: how long to cache the IP. Use values like `30s`, `5m`, or `1h`. Use `none` to disable caching

## Template

### Default Template

```template
 {{ .IP }}
```

### Properties

- `.IP`
  - Type: `string`
  - Description: Your external IP address

[ipify]: https://www.ipify.org/
