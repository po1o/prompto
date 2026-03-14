# Execution Time

## Segment Type

`executiontime`

## What

Displays the execution time of the previously executed command.

## Sample Configuration

```yaml
prompt:
  - segments: ["executiontime"]

executiontime:
  type: "executiontime"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#8800dd"
  template: " <#fefefe></> {{ .FormattedMs }} "
  options:
    threshold: 500
    style: "austin"
    always_enabled: true
```

## Options

- `always_enabled`
  - Type: `boolean`
  - Default: `false`
  - Description: always show the duration
- `threshold`
  - Type: `int`
  - Default: `500`
  - Description: minimum duration (milliseconds) required to enable this segment
- `style`
  - Type: `enum`
  - Default: `austin`
  - Description: one of the available format options

### Style

Style specifies the format in which the time will be displayed. The table below shows some example times in each option.

- `austin`
  - 0.001s: `1ms`
  - 2.1s: `2.1s`
  - 3m2.1s: `3m 2.1s`
  - 4h3m2.1s: `4h 3m 2.1s`
- `roundrock`
  - 0.001s: `1ms`
  - 2.1s: `2s 100ms`
  - 3m2.1s: `3m 2s 100ms`
  - 4h3m2.1s: `4h 3m 2s 100ms`
- `dallas`
  - 0.001s: `0.001`
  - 2.1s: `2.1`
  - 3m2.1s: `3:2.1`
  - 4h3m2.1s: `4:3:2.1`
- `galveston`
  - 0.001s: `00:00:00`
  - 2.1s: `00:00:02`
  - 3m2.1s: `00:03:02`
  - 4h3m2.1s: `04:03:02`
- `galvestonms`
  - 0.001s: `00:00:00:001`
  - 2.1s: `00:00:02:100`
  - 3m2.1s: `00:03:02:100`
  - 4h3m2.1s: `04:03:02:100`
- `houston`
  - 0.001s: `00:00:00.001`
  - 2.1s: `00:00:02.1`
  - 3m2.1s: `00:03:02.1`
  - 4h3m2.1s: `04:03:02.1`
- `amarillo`
  - 0.001s: `0.001s`
  - 2.1s: `2.1s`
  - 3m2.1s: `182.1s`
  - 4h3m2.1s: `14,582.1s`
- `round`
  - 0.001s: `1ms`
  - 2.1s: `2s`
  - 3m2.1s: `3m 2s`
  - 4h3m2.1s: `4h 3m`
- `lucky7`
  - 0.001s: `1ms`
  - 2.1s: ` 2.00s `
  - 3m2.1s: `3m 2s`
  - 4h3m2.1s: `4h 3m`

## Template

### Default Template

```template
 {{ .FormattedMs }}
```

### Properties

- `.Ms`
  - Type: `number`
  - Description: the execution time in milliseconds
- `.FormattedMs`
  - Type: `string`
  - Description: the formatted value based on the `style` above
