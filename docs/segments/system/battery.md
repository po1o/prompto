# Battery

## Segment Type

`battery`

## What

### Caution

The segment is not supported and automatically disabled on Windows when WSL 1 is detected. Works fine with WSL 2.

Battery displays the remaining power percentage for your battery.

## Sample Configuration

```yaml
prompt:
  - segments: ["battery"]

battery:
  type: "battery"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#193549"
  background: "#ffeb3b"
  background_templates: ["{{if eq \"Charging\" .State.String}}#40c4ff{{end}}", "{{if eq \"Discharging\" .State.String}}#ff5722{{end}}", "{{if eq \"Full\" .State.String}}#4caf50{{end}}"]
  template: " {{ if not .Error }}{{ .Icon }}{{ .Percentage }}{{ end }} "
  options:
    discharging_icon: " "
    charging_icon: " "
    charged_icon: " "
```

## Options

- `display_error`
  - Type: `boolean`
  - Default: `false`
  - Description: show the error context when failing to retrieve the battery information
- `charging_icon`
  - Type: `string`
  - Description: icon to display when charging
- `discharging_icon`
  - Type: `string`
  - Description: icon to display when discharging
- `charged_icon`
  - Type: `string`
  - Description: icon to display when fully charged
- `not_charging_icon`
  - Type: `string`
  - Description: icon to display when fully charged

## Template

### Default Template

```template
 {{ if not .Error }}{{ .Icon }}{{ .Percentage }}{{ end }}{{ .Error }}
```

### Properties

- `.State`
  - Type: `struct`
  - Description: the battery state, has a `.String` function
- `.Current`
  - Type: `float64`
  - Description: Current (momentary) charge rate (in mW).
- `.Full`
  - Type: `float64`
  - Description: Last known full capacity (in mWh)
- `.Design`
  - Type: `float64`
  - Description: Reported design capacity (in mWh)
- `.ChargeRate`
  - Type: `float64`
  - Description: Current (momentary) charge rate (in mW). It is always non-negative, consult .State field to check
    whether it means charging or discharging (on some systems this might be always `0` if the battery doesn't support
    it)
- `.Voltage`
  - Type: `float64`
  - Description: Current voltage (in V)
- `.DesignVoltage`
  - Type: `float64`
  - Description: Design voltage (in V). Some systems (e.g. macOS) do not provide a separate value for this. In such
    cases, or if getting this fails, but getting `Voltage` succeeds, this field will have the same value as `Voltage`,
    for convenience
- `.Percentage`
  - Type: `float64`
  - Description: the current battery percentage
- `.Error`
  - Type: `string`
  - Description: the error in case fetching the battery information failed
- `.Icon`
  - Type: `string`
  - Description: the icon based on the battery state
