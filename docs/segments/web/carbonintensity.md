# Carbon Intensity

## Segment Type

`carbonintensity`

## What

Shows the actual and forecast carbon intensity in gCO2/kWh using data from the [Carbon Intensity
API][carbonintensity-api].

### Note

Note that this segment only provides data for Great Britain at the moment. Support for other countries may become
available in the future.

## Sample Configuration

### Caution

The API can be slow. It's recommended to set the `http_timeout` property to a large value (e.g. `5000`).

```yaml
prompt:
  - segments: ["carbonintensity"]

carbonintensity:
  type: "carbonintensity"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#000000"
  background: "#ffffff"
  background_templates:
    - "{{if eq \"very low\" .Index}}#a3e635{{end}}"
    - "{{if eq \"low\" .Index}}#bef264{{end}}"
    - "{{if eq \"moderate\" .Index}}#fbbf24{{end}}"
    - "{{if eq \"high\" .Index}}#ef4444{{end}}"
    - "{{if eq \"very high\" .Index}}#dc2626{{end}}"
  template: " CO₂ {{ .Index.Icon }}{{ .Actual.String }} {{ .TrendIcon }} {{ .Forecast.String }} "
  options:
    http_timeout: 5000
```

## Options

- `http_timeout`
  - Type: `int`
  - Default: `20`
  - Description: Timeout (in milliseconds) for HTTP requests. The default is 20ms, but you may need to set this to as
    high as 5000ms to handle slow API requests.

## Template

### Default Template

```template
 CO₂ {{ .Index.Icon }}{{ .Actual.String }} {{ .TrendIcon }} {{ .Forecast.String }}
```

### Properties

- `.Forecast`
  - Type: `Number`
  - Description: The forecast carbon intensity in gCO2/kWh. Equal to `0` if no data is available.
- `.Actual`
  - Type: `Number`
  - Description: The actual carbon intensity in gCO2/kWh. Equal to `0` if no data is available.
- `.Index`
  - Type: `Index`
  - Description: A rating of the current carbon intensity. Possible values are `"very low"`, `"low"`, `"moderate"`,
    `"high"`, or `"very high"`. Equal to `"??"` if no data is available.
- `.TrendIcon`
  - Type: `string`
  - Description: An icon representation of the predicted trend in carbon intensity based on the Actual and Forecast
    values. Possible values are `"↗"`, `"↘"`, or `"→"`.

#### Number

- `.String`
  - Type: `string`
  - Description: string representation of the value

#### Index

- `.Icon`
  - Type: `string`
  - Description: icon representation of the value

[carbonintensity-api]: https://carbon-intensity.github.io/api-definitions
