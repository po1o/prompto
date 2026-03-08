---
title: Unity
description: Display the currently active Unity and C# versions.
---

## Segment Type

`unity`

## What

Display the currently active [Unity][unity] and C# versions.

The Unity version is displayed regardless of whether or not the corresponding C# version can be found. The C# version is
determined by first checking a static table. If the Unity version isn't found, a web request is made to [the Unity
docs][unity-csharp-page] to try extracting it from there. A web request only occurs the first time a given `major.minor`
Unity version is encountered. Subsequent invocations return the cached C# version.

C# version display is only supported from Unity 2017.1.

Unity 2017.1 - 2019.1 support two C# versions, depending on which scripting runtime is selected in Player Settings. This
segment always chooses the higher version.

## Sample Configuration

```yaml
prompt:
  - segments: ["unity"]

unity:
  type: "unity"
  style: "powerline"
  powerline_symbol: ""
  foreground: "#111111"
  background: "#ffffff"
  options:
    http_timeout: 2000
```

## Options

- `http_timeout`
  - Type: `int`
  - Default: `2000`
  - Description: in milliseconds - the timeout for http request

## Template

### Default Template

```template
\ue721 {{ .UnityVersion }}{{ if .CSharpVersion }} {{ .CSharpVersion }}{{ end }}
```

### Properties

- `.UnityVersion`
  - Type: `string`
  - Description: the Unity version
- `.CSharpVersion`
  - Type: `string`
  - Description: the C# version

[unity]: https://unity.com/
[unity-csharp-page]: https://docs.unity3d.com/Manual/CSharpCompiler.html
