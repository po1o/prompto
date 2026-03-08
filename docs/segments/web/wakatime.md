---
title: Wakatime
description: Shows the tracked time on wakatime of the current day
---

## Segment Type

`wakatime`

## What

Shows the tracked time on [wakatime][wt] of the current day

### Caution

You **must** request an API key at the [wakatime][wt] website. The free tier for is sufficient. You'll find the API key
in your profile settings page.

## Sample Configuration

```yaml
prompt:
  - segments: ["wakatime"]

wakatime:
  type: "wakatime"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#007acc"
  options:
    url: "https://wakatime.com/api/v1/users/current/summaries?start=today&end=today&api_key=API_KEY"
    http_timeout: 500
```

## Options

- `url`
  - Type: `string`
  - Description: The Wakatime [summaries][wk-summaries] URL, including the API key. Example above.
- `http_timeout`
  - Type: `int`
  - Default: `20`
  - Description: The time (_in milliseconds_, `ms`) it takes to consider an http request as **timed-out**. If no segment
    is shown, try increasing this timeout.

### Dynamic API Key

If you don't want to include the API key into your configuration, the following modification can be done.

```yaml
type: "wakatime"
options:
  url: "https://wakatime.com/api/v1/users/current/summaries?start=today&end=today&api_key={{ .Env.WAKATIME_API_KEY }}"
  http_timeout: 500
```

### Note

`WAKATIME_API_KEY` is an example, **any name is possible and acceptable** as long as the environment variable exists and
contains the API key value.

Please refer to the [Environment Variable][templates-environment-variables] page for more information.

## Template

### Default Template

```template
 {{ secondsRound .CumulativeTotal.Seconds }}
```

### Properties

- `.CumulativeTotal`
  - Type: `wtTotals`
  - Description: object holding total tracked time values

### wtTotals Properties

- `.Seconds`
  - Type: `float64`
  - Description: a number representing the total tracked time in seconds
- `.Text`
  - Type: `string`
  - Description: a string with human readable tracked time (eg: "2 hrs 30 mins")

[wt]: https://wakatime.com
[wk-summaries]: https://wakatime.com/developers#summaries
[templates-environment-variables]: ../../configuration/templates.md#environment-variables
