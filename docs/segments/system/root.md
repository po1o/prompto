# Root

## Segment Type

`root`

## What

Show when the current user is root or when in an elevated shell (Windows).

## Sample Configuration

```yaml
prompt:
  - segments: ["root"]

root:
  type: "root"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#111111"
  background: "#ffff66"
  template: ""
```

## Template

### Default Template

```template
 \uF0E7
```
