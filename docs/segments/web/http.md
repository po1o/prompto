# HTTP

## Segment Type

`http`

## What

HTTP Request is a simple segment to return any json data from any HTTP call.

## Sample Configuration

```yaml
prompt:
  - segments: ["http"]

http:
  type: "http"
  style: "diamond"
  foreground: "#ffffff"
  background: "#c386f1"
  leading_diamond: ""
  trailing_diamond: ""
  template: "{{ .Result }}"
  options:
    url: "https://jsonplaceholder.typicode.com/posts/1"
    method: "GET"
```

## Options

- `url`
  - Type: `string`
  - Default: ``
  - Description: The HTTP URL you want to call, supports [templates]
- `method`
  - Type: `string`
  - Default: `GET`
  - Description: The HTTP method to use, `GET` or `POST`

## Template

### Default Template

```template
 {{ .Body }}
```

### Properties

- `.Body.property`
  - Type: `string`
  - Description: Replace `.property` with the property you want to display

[templates]: ../../configuration/templates.md
