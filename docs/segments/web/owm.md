---
title: Open Weather Map
description: Shows the current weather of a given location with Open Weather Map.
---

## Segment Type

`owm`

## What

Shows the current weather of a given location with [Open Weather Map][owm].

### Caution

You **must** request an API key at the [Open Weather Map][owm-price] website. The free tier for _Current weather and
forecasts collection_ is sufficient.

## Sample Configuration

```yaml
prompt:
  - segments: ["owm"]

owm:
  type: "owm"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#FF0000"
  template: "{{.Weather}} ({{.Temperature}}{{.UnitIcon}})"
  options:
    api_key: "<YOUR_API_KEY>"
    location: "AMSTERDAM,NL"
    units: "metric"
    http_timeout: 20
```

## Options

- `api_key`
  - Type: [`template`][templates]
  - Default: `.`
  - Description: Your API key from [Open Weather Map][owm].
- `location`
  - Type: [`template`][templates]
  - Default: `De Bilt,NL`
  - Description: The requested location interpreted only if valid coordinates aren't given. Formatted as \. City name,
    state code and country code divided by comma. Please, refer to ISO 3166 for the state codes or country codes .
- `units`
  - Type: `string`
  - Default: `standard`
  - Description: Units of measurement. Available values are standard (kelvin), metric (celsius), and imperial
    (fahrenheit)
- `http_timeout`
  - Type: `int`
  - Default: `20`
  - Description: in milliseconds, the timeout for http request

## Template

### Default Template

```template
 {{ .Weather }} ({{ .Temperature }}{{ .UnitIcon }})
```

### Properties

- `.Weather`
  - Type: `string`
  - Description: the current weather icon
- `.Temperature`
  - Type: `int`
  - Description: the current temperature
- `.UnitIcon`
  - Type: `string`
  - Description: the current unit icon(based on units property)
- `.URL`
  - Type: `string`
  - Description: the url of the current api call

[templates]: ../../configuration/templates.md
[owm]: https://openweathermap.org
[owm-price]: https://openweathermap.org/price
