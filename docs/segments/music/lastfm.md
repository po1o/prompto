---
title: LastFM
description: Show the currently playing song from a LastFM user.
---

## Segment Type

`lastfm`

## What

Show the currently playing song from a [LastFM][lastfm] user.

### Caution

Be aware that LastFM updates may be severely delayed when paused and songs may linger in the "now playing" state for a
prolonged time.

Additionally, we are using HTTP requests to get the data, so you may need to adjust the `http_timeout` to your liking to
get better results.

You **must** request an [API key][api-key] at the LastFM website.

## Sample Configuration

```yaml
prompt:
  - segments: ["lastfm"]

lastfm:
  background: "p:sky"
  foreground: "p:white"
  powerline_symbol: ""
  options:
    api_key: "<YOUR_API_KEY>"
    username: "<LASTFM_USERNAME>"
    http_timeout: 20000
  style: "powerline"
  template: " {{ .Icon }}{{ if ne .Status \"stopped\" }}{{ .Full }}{{ end }} "
  type: "lastfm"
```

## Options

- `playing_icon`
  - Type: `string`
  - Default: `\uE602`
  - Description: text/icon to show when playing
- `stopped_icon`
  - Type: `string`
  - Default: `\uF04D`
  - Description: text/icon to show when stopped
- `api_key`
  - Type: [`template`][templates]
  - Default: `.`
  - Description: your LastFM [API key][api-key]
- `username`
  - Type: `string`
  - Default: `.`
  - Description: your LastFM username
- `http_timeout`
  - Type: `int`
  - Default: `20`
  - Description: in milliseconds - the timeout for http request

## Template

### Default Template

```template
{{ .Icon }}{{ if ne .Status \"stopped\" }}{{ .Full }}{{ end }}
```

### Properties

- `.Status`
  - Type: `string`
  - Description: player status (`playing`, `paused`, `stopped`)
- `.Artist`
  - Type: `string`
  - Description: current artist
- `.Track`
  - Type: `string`
  - Description: current track
- `.Full`
  - Type: `string`
  - Description: will output `Artist - Track`
- `.Icon`
  - Type: `string`
  - Description: icon (based on `.Status`)

[templates]: ../../configuration/templates.md
[lastfm]: https://www.last.fm
[api-key]: https://www.last.fm/api/account/create
