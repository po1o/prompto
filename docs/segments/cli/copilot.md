---
title: GitHub Copilot
description: Display your GitHub Copilot usage statistics and quota information including premium interactions, inline completions, and chat usage. This segment was inspired by Elio Struyf's GitHub Copilot Usage Tauri application.
---

## Segment Type

`copilot`

## What

Display your [GitHub Copilot][copilot] usage statistics and quota information including premium interactions, inline
completions, and chat usage. This segment was inspired by [Elio Struyf's GitHub Copilot Usage Tauri application][tauri].

## Authentication

This segment requires authentication with GitHub to access Copilot usage data. Use the built-in OAuth device code flow:

```bash
prompto auth copilot
```

This will:

1. Display a device code and verification URL
2. Open your browser to GitHub's authorization page
3. Prompt you to enter the device code
4. Store the access token securely for future use

The token is stored securely and will be used automatically by the segment.

## Sample Configuration

```yaml
prompt:
  - segments: ["copilot"]

copilot:
  type: "copilot"
  style: "diamond"
  leading_diamond: ""
  trailing_diamond: ""
  foreground: "#111111"
  background: "#fee898"
  template: "  {{ .Premium.Percent.Gauge }} "
  cache:
    duration: "5m"
    strategy: "session"
  options:
    http_timeout: 1000
```

## Options

- `http_timeout`
  - Type: `int`
  - Default: `20`
  - Description: The default timeout for HTTP requests in milliseconds

## Template

### Default Template

```template
 \uec1e {{ .Premium.Percent.Gauge }}
```

### Properties

- `.Premium`
  - Type: `CopilotUsage`
  - Description: Premium interactions usage data
- `.Premium.Used`
  - Type: `int`
  - Description: Number of premium interactions used
- `.Premium.Limit`
  - Type: `int`
  - Description: Total premium interactions available
- `.Premium.Percent`
  - Type: `Percentage`
  - Description: Percentage of premium quota used (0-100)
- `.Premium.Remaining`
  - Type: `Percentage`
  - Description: Percentage of premium quota remaining (0-100)
- `.Premium.Unlimited`
  - Type: `bool`
  - Description: Whether premium quota is unlimited
- `.Inline`
  - Type: `CopilotUsage`
  - Description: Inline completions usage data
- `.Inline.Used`
  - Type: `int`
  - Description: Number of inline completions used
- `.Inline.Limit`
  - Type: `int`
  - Description: Total inline completions available
- `.Inline.Percent`
  - Type: `Percentage`
  - Description: Percentage of inline quota used (0-100)
- `.Inline.Remaining`
  - Type: `Percentage`
  - Description: Percentage of inline quota remaining (0-100)
- `.Inline.Unlimited`
  - Type: `bool`
  - Description: Whether inline quota is unlimited
- `.Chat`
  - Type: `CopilotUsage`
  - Description: Chat usage data
- `.Chat.Used`
  - Type: `int`
  - Description: Number of chat interactions used
- `.Chat.Limit`
  - Type: `int`
  - Description: Total chat interactions available
- `.Chat.Percent`
  - Type: `Percentage`
  - Description: Percentage of chat quota used (0-100)
- `.Chat.Remaining`
  - Type: `Percentage`
  - Description: Percentage of chat quota remaining (0-100)
- `.Chat.Unlimited`
  - Type: `bool`
  - Description: Whether chat quota is unlimited
- `.BillingCycleEnd`
  - Type: `string`
  - Description: End date of current billing cycle

### Percentage Methods

The `Percentage` type provides additional functionality beyond just the numeric value:

- `.Gauge()`
  - Returns: `string`
  - Description: Visual gauge showing remaining capacity using 5 bar blocks (▰▰▰▰▱)
- `.GaugeUsed()`
  - Returns: `string`
  - Description: Visual gauge showing used capacity using 5 bar blocks (▰▱▱▱▱)
- `.String()`
  - Returns: `string`
  - Description: Numeric percentage value (e.g., "75" for use in templates)

**Example gauge visualization (shows remaining capacity):**

- 0% used (100% remaining): `▰▰▰▰▰`
- 20% used (80% remaining): `▰▰▰▰▱`
- 40% used (60% remaining): `▰▰▰▱▱`
- 60% used (40% remaining): `▰▰▱▱▱`
- 80% used (20% remaining): `▰▱▱▱▱`
- 100% used (0% remaining): `▱▱▱▱▱`

**Example gaugeUsed visualization (shows used capacity):**

- 0% used: `▱▱▱▱▱`
- 20% used: `▰▱▱▱▱`
- 40% used: `▰▰▱▱▱`
- 60% used: `▰▰▰▱▱`
- 80% used: `▰▰▰▰▱`
- 100% used: `▰▰▰▰▰`

**Example template with gauge:**

```json
"template": "{{ .Premium.Percent.Gauge() }} {{ .Premium.Used }}/{{ .Premium.Limit }}"
```

**Example template showing used capacity:**

```json
"template": "{{ .Premium.Percent.GaugeUsed() }} {{ .Premium.Used }}/{{ .Premium.Limit }}"
```

[copilot]: https://github.com/features/copilot
[tauri]: https://github.com/estruyf/github-copilot-usage-tauri
