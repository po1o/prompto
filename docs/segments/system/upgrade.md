---
title: Upgrade Notice
description: Display when an update is available for prompto.
---

## Segment Type

`upgrade`

## What

Display when an update is available for `prompto`.

## Sample Configuration

```yaml
prompt:
  - segments: ["upgrade"]

upgrade:
  type: "upgrade"
  style: "plain"
  foreground: "#111111"
  background: "#FFD664"
  options:
    cache_duration: "168h"
```

## Options

- `cache_duration`
  - Type: `string`
  - Default: `168h`
  - Description: the duration for which the segment will be cached. The duration is a string in the format `1h2m3s` and
    is parsed using the [time.ParseDuration] function from the Go standard library. To disable the cache, use `none`

## Template

### Default Template

```template
 \uf019
```

### Properties

- `.Current`
  - Type: `string`
  - Description: the current version number
- `.Latest`
  - Type: `string`
  - Description: the latest available version number

[time.ParseDuration]: https://golang.org/pkg/time/#ParseDuration
