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
{{ .LastDate | date .Format }}
```

### Properties

- `.Format`
  - Type: `string`
  - Description: The time format resolved from `time_format`.
- `.LastDate`
  - Type: `time`
  - Description: The time the previous command finished.
- `.CurrentDate`
  - Type: `string`
  - Description: The time the current command started.

## Behavior Difference

The important difference is what the timestamp refers to:

- `.LastDate`: the time the previous command finished
- `.CurrentDate`: the time the current command started

By prompt type:

| Prompt type | `.LastDate` | `.CurrentDate` |
| --- | --- | --- |
| `prompt`, `rprompt` | previous command finished | previous command finished, because there is no current command yet |
| `transient`, `rtransient` | previous command finished | current command started |

So:

- use `.LastDate` when you want to show when the previous command finished
- use `.CurrentDate` when you want transient prompts to show when the current command started

### Example: `sleep 3m`

Assume this timeline:

1. Your previous command finishes at `10:00:00`
2. You wait on the prompt for two minutes
3. At `10:02:00` you press Enter on `sleep 3m`
4. The command finishes at `10:05:00`

While you are typing `sleep 3m`, this is still a normal `prompt` / `rprompt`.
There is no current command yet, so both properties show the same time:

```text
.LastDate    -> 10:00:00
.CurrentDate -> 10:00:00
```

When you press Enter, the transient prompt is shown for the command you just started:

```text
.LastDate    -> 10:00:00
.CurrentDate -> 10:02:00
```

After `sleep 3m` finishes at `10:05:00`, the next normal prompt is shown.
Again there is no current command yet, so both properties match:

```text
.LastDate    -> 10:05:00
.CurrentDate -> 10:05:00
```

## Choosing The Property

The default template uses `.LastDate`, so the timestamp is rendered once and stays fixed.

If you want transient prompts to show when the current command started, use `.CurrentDate` in your template:

```yaml
time:
  type: time
  template: " {{ .CurrentDate }} "
  options:
    time_format: "15:04:05"
```

This distinction matters most for transient prompts, because they are shown after you press Enter.

## Format Rules

`.LastDate` and `.CurrentDate` share the same supported `time_format`.

Supported Go layout tokens:

- `2006`, `06`
- `January`, `Jan`
- `01`
- `02`, `_2`, `2`
- `Monday`, `Mon`
- `15`, `03`, `3`
- `04`
- `05`
- `PM`
- `MST`
- `-0700`

Plain literal text and separators such as spaces, `:`, `-`, `/`, `@`, `|`, `(`, `)`, and icons are also allowed.

Prompt markup is not part of `time_format`.
If you want colored text such as `<#fff>at</>`, put it in the segment template instead.

Supported predefined constants:

- `Layout`
- `ANSIC`
- `UnixDate`
- `RubyDate`
- `RFC822`
- `RFC822Z`
- `RFC850`
- `RFC1123`
- `RFC1123Z`
- `Kitchen`
- `Stamp`
- `DateTime`
- `DateOnly`
- `TimeOnly`

Unsupported formats fail config validation.

Common unsupported examples:

- `1`, `4`, `5`
- `pm`
- fractional seconds such as `.000` or `.999999999`
- timezone forms `-07`, `-07:00`, `Z0700`, `Z07:00`
- predefined formats such as `RFC3339`, `RFC3339Nano`, `StampMilli`, `StampMicro`, `StampNano`
- prompt markup such as `<#fff>...`

## Syntax

### Supported Tokens

- **Year**
  - Format: `06`, `2006`
- **Month**
  - Format: `01`, `Jan`, `January`
- **Day**
  - Format: `02`, `_2` (width two, right justified), `2`
- **Weekday**
  - Format: `Mon`, `Monday`
- **Hours**
  - Format: `03`, `3`, `15`
- **Minutes**
  - Format: `04`
- **Seconds**
  - Format: `05`
- **am/pm**
  - Format: `PM`
- **Timezone**
  - Format: `MST`
- **Offset**
  - Format: `-0700`

Other plain characters in `time_format` are treated as literal text.

### Predefined Formats

The following predefined date and timestamp [format constants][format-constants] are supported:

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
- **Kitchen**
  - Format: `3:04PM`
- **Stamp**
  - Format: `Jan _2 15:04:05`
- **DateTime**
  - Format: `2006-01-02 15:04:05`
- **DateOnly**
  - Format: `2006-01-02`
- **TimeOnly**
  - Format: `15:04:05`

## Examples

To display the time in multiple time zones, using [Sprig's Date Functions][sprig-date]:

```text
{{ .LastDate | date .Format }} {{ dateInZone "15:04Z" .LastDate "UTC" }}
```

To display the time the current command started in a transient prompt:

```yaml
transient:
  - segments: ["time"]

time:
  type: time
  template: " {{ .CurrentDate }} "
  options:
    time_format: "15:04:05"
```

[format-constants]: https://golang.org/pkg/time/#pkg-constants
[sprig-date]: https://masterminds.github.io/sprig/date.html
