---
title: Time
description: Show the current timestamp.
---

## Segment Type

`time`

## What

Show the current timestamp.

## Sample Configuration

```yaml
prompt:
  - segments: ["time"]

time:
  type: "time"
  style: "plain"
  foreground: "#007ACC"
  options:
    time_format: "15:04:05"
```

## Options

- `time_format`
  - Type: `string`
  - Default: `15:04:05`
  - Description: Format to use

## Template

### Default Template

```template
 {{ .CurrentDate | date .Format }}
```

### Properties

- `.Format`
  - Type: `string`
  - Description: The time format (set via `time_format`)
- `.CurrentDate`
  - Type: `time`
  - Description: The time to display (testing purpose)

## Syntax

### Formats

Follows the [golang datetime standard][format]:

- **Year**
  - Format: `06`, `2006`
- **Month**
  - Format: `01`, `1`, `Jan`, `January`
- **Day**
  - Format: `02`, `2`, `_2` (width two, right justified)
- **Weekday**
  - Format: `Mon`, `Monday`
- **Hours**
  - Format: `03`, `3`, `15`
- **Minutes**
  - Format: `04`, `4`
- **Seconds**
  - Format: `05`, `5`
- **ms μs ns**
  - Format: `.000`, `.000000`, `.000000000`
- **ms μs ns** (trailing zeros removed)
  - Format: `.999`, `.999999`, `.999999999`
- **am/pm**
  - Format: `PM`, `pm`
- **Timezone**
  - Format: `MST`
- **Offset**
  - Format: `-0700`, `-07`, `-07:00`, `Z0700`, `Z07:00`

### Predefined formats

The following predefined date and timestamp [format constants][format-constants] are also available:

- **ANSIC**
  - Format: `Mon Jan _2 15:04:05 2006`
- **UnixDate**
  - Format: `Mon Jan _2 15:04:05 MST 2006`
- **RubyDate**
  - Format: `Mon Jan 02 15:04:05 -0700 2006`
- **RFC822**
  - Format: `02 Jan 06 15:04 MST`
- **RFC822Z**
  - Format: `02 Jan 06 15:04 -0700`
- **RFC850**
  - Format: `Monday, 02-Jan-06 15:04:05 MST`
- **RFC1123**
  - Format: `Mon, 02 Jan 2006 15:04:05 MST`
- **RFC1123Z**
  - Format: `Mon, 02 Jan 2006 15:04:05 -0700`
- **RFC3339**
  - Format: `2006-01-02T15:04:05Z07:00`
- **RFC3339Nano**
  - Format: `2006-01-02T15:04:05.999999999Z07:00`
- **Kitchen**
  - Format: `3:04PM`
- **Stamp**
  - Format: `Jan _2 15:04:05`
- **StampMilli**
  - Format: `Jan _2 15:04:05.000`
- **StampMicro**
  - Format: `Jan _2 15:04:05.000000`
- **StampNano**
  - Format: `Jan _2 15:04:05.000000000`

## Examples

To display the time in multiple time zones, using [Sprig's Date Functions][sprig-date]:

```text
{{ .CurrentDate | date .Format }} {{ dateInZone "15:04Z" .CurrentDate "UTC" }}
```

[format]: https://yourbasic.org/golang/format-parse-string-time-date-example/
[format-constants]: https://golang.org/pkg/time/#pkg-constants
[sprig-date]: https://masterminds.github.io/sprig/date.html
