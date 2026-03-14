# Todoist

## Segment Type

`todoist`

## What

Displays your daily tasks from [Todoist][todoist].

### Caution

The segment needs an [API Key][guide] from your Todoist profile for this to work.

## Sample Configuration

```yaml
prompt:
  - segments: ["todoist"]

todoist:
  type: "todoist"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#FF0000"
  template: "{{.TaskCount}}"
  options:
    api_key: "<YOUR_API_KEY>"
    http_timeout: 500
```

## Options

- `api_key`
  - Type: `string`
  - Default: `.`
  - Description: Your API Key from [Todoist][todoist]
- `http_timeout`
  - Type: `int`
  - Default: `20`
  - Description: The time (_in milliseconds_, `ms`) it takes to consider an http request as **timed-out**. If no segment
    is shown, try increasing this timeout.

## Template

### Default Template

```template
 {{ .TaskCount }}
```

### Properties

- `.TaskCount`
  - Type: `int`
  - Description: the number of tasks due today

[todoist]: https://www.todoist.com/
[guide]: https://www.todoist.com/help/articles/find-your-api-token-Jpzx9IIlB
