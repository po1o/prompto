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
  - Description: Format to use. This follows Go time layouts and the predefined names listed below.

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
- `.ShellClock`
  - Type: `string`
  - Description: A shell-native live clock string that follows `time_format` when the format can be translated
    exactly to a portable `strftime` subset. Otherwise it falls back to the already-rendered timestamp string.

## Shell Clock

The default template uses `.CurrentDate`, so the timestamp is rendered once per prompt.

If you want the shell to keep the clock live without re-rendering the prompt, use `.ShellClock` in your template:

```yaml
time:
  type: time
  template: " {{ .ShellClock }} "
  options:
    time_format: "15:04:05"
```

Behavior by shell:

- `zsh`: emits `%D{...}`
- `bash`: emits `\D{...}`
- `fish` and `pwsh`: emits a placeholder that the init script expands at display time
- other shells: fall back to a normal rendered string

`.ShellClock` only uses the shell-native live clock path when `time_format` can be translated exactly.
If not, `.ShellClock` falls back to the same rendered value you would get from `.CurrentDate | date .Format`.

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

### Formats That `.ShellClock` Can Follow Exactly

The live shell clock path supports these Go layout tokens exactly, plus literal separators such as `:`, `-`, `/`,
spaces, and `T`:

- `2006`, `06`
- `January`, `Jan`
- `01`
- `02`, `_2`
- `Monday`, `Mon`
- `15`, `03`
- `04`
- `05`
- `PM`
- `MST`
- `-0700`

These formats therefore work well with `.ShellClock`:

- `15:04:05`
- `2006-01-02 15:04:05`
- `Mon Jan _2 15:04:05 MST 2006`
- `DateTime`
- `DateOnly`
- `TimeOnly`

These do not have an exact portable shell-clock translation and therefore fall back to a rendered string:

- `1`, `2`, `3`, `4`, `5`
- `pm`
- fractional seconds such as `.000` or `.999999999`
- timezone forms `-07`, `-07:00`, `Z0700`, `Z07:00`
- predefined formats such as `Kitchen`, `RFC3339`, `RFC3339Nano`, `StampMilli`, `StampMicro`, `StampNano`

### Predefined formats

The following predefined date and timestamp [format constants][format-constants] are also available:

- **Layout**
  - Format: `01/02 03:04:05PM '06 -0700`
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
- **DateTime**
  - Format: `2006-01-02 15:04:05`
- **DateOnly**
  - Format: `2006-01-02`
- **TimeOnly**
  - Format: `15:04:05`

## Examples

To display the time in multiple time zones, using [Sprig's Date Functions][sprig-date]:

```text
{{ .CurrentDate | date .Format }} {{ dateInZone "15:04Z" .CurrentDate "UTC" }}
```

[format]: https://yourbasic.org/golang/format-parse-string-time-date-example/
[format-constants]: https://golang.org/pkg/time/#pkg-constants
[sprig-date]: https://masterminds.github.io/sprig/date.html
