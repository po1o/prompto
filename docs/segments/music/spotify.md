---
title: Spotify
description: Show the currently playing song in the Spotify client.
---

## Segment Type

`spotify`

## What

Show the currently playing song in the [Spotify][spotify] client.

### Caution

Be aware this can make the prompt a tad bit slower as it needs to get a response from the Spotify player.

On _macOS & Linux_, all states are supported (playing/paused/stopped).

On _Windows/WSL_, **only the playing state is supported** (no information when paused/stopped). It supports fetching
information from the native Spotify application and Edge PWA.

## Sample Configuration

```yaml
prompt:
  - segments: ["spotify"]

spotify:
  type: "spotify"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#ffffff"
  background: "#1BD760"
  options:
    playing_icon: " "
    paused_icon: " "
    stopped_icon: " "
```

## Options

- `playing_icon`
  - Type: `string`
  - Default: `\ue602`
  - Description: text/icon to show when playing
- `paused_icon`
  - Type: `string`
  - Default: `\uf04c`
  - Description: text/icon to show when paused
- `stopped_icon`
  - Type: `string`
  - Default: `\uf04d`
  - Description: text/icon to show when stopped

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

[spotify]: https://www.spotify.com
