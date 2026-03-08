---
title: YouTube Music
description: Shows the currently playing song in the YouTube Music Desktop App.
---

## Segment Type

`ytm`

## What

Shows the currently playing song in the [YouTube Music Desktop App][ytmdesktop].

## Setup

You need to enable the Companion API in the YouTube Music Desktop App settings. To do this, open the app, go to
`Settings > Integration` and enable the following:

- Companion server
- Enable companion authentication

From the CLI, run the following command to set the authentication token:

```bash
prompto auth ytmda
```

If done correctly, you should now be able to add the `ytm` segment to your prompt.

### Rate Limiting

The YouTube Music Desktop App has a pretty strict rate limit. Therefore it is recommended to set the `cache` property in
your configuration. If you don't, the segment will not be able to display correctly.

## Sample Configuration

```yaml
prompt:
  - segments: ["ytm"]

ytm:
  type: "ytm"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#FF0000"
  options:
    playing_icon: " "
    paused_icon: " "
    stopped_icon: " "
    ad_icon: " "
    http_timeout: 1000
  cache:
    duration: "5s"
    strategy: "session"
```

## Options

- `playing_icon`
  - Type: `string`
  - Default: `\uf04b`
  - Description: text/icon to show when playing
- `paused_icon`
  - Type: `string`
  - Default: `\uf04c`
  - Description: text/icon to show when paused
- `stopped_icon`
  - Type: `string`
  - Default: `\uf04d`
  - Description: text/icon to show when stopped
- `ad_icon`
  - Type: `string`
  - Default: `\ueebb`
  - Description: text/icon to show when an advertisement is playing
- `http_timeout`
  - Type: `int`
  - Default: `5000`
  - Description: in milliseconds - the timeout for http request

## Template

### Default Template

```template
{{ .Icon }}{{ if ne .Status \"stopped\" }}{{ .Artist }} - {{ .Track }}{{ end }}
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
- `.Icon`
  - Type: `string`
  - Description: icon (based on `.Status`)

[ytmdesktop]: https://github.com/ytmdesktop/ytmdesktop
