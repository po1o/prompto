---
title: Configuration Quick Start
description: Build a prompto YAML configuration step by step, starting from a minimal prompt.
---

## Goal

This page walks from a minimal prompt to a practical multi-part layout with a right prompt and transient prompt.

## Step 1: Start with a Single Segment

```yaml
cursor_padding: true

prompt:
  - segments: [path]

path:
  foreground: white
  background: blue
  template: " {{ .Path }} "
```

This gives you a single left prompt line containing the current path.

## Step 2: Add a Right Prompt

```yaml
prompt:
  - segments: [session, path]

rprompt:
  - segments: [git, time]

session:
  foreground: black
  background: yellow
  template: " {{ if .SSHSession }} {{ .UserName }}@{{ .HostName }} {{ else }} {{ .UserName }} {{ end }}"

path:
  foreground: white
  background: blue
  template: " {{ .Path }} "

git:
  foreground: black
  background_templates:
    - "{{ if or (.Working.Changed) (.Staging.Changed) }}yellow{{ else }}green{{ end }}"
  template: " {{ .HEAD }} "
  options:
    fetch_status: true

time:
  foreground: white
  background: darkGray
  template: " {{ .CurrentDate | date \"15:04\" }} "
```

## Step 3: Add Layout Styling

Line-level `style` is a separator shortcut.
For a left prompt line it becomes the trailing separator.
For a right prompt line it becomes the leading separator.

```yaml
prompt:
  - style: rounded
    segments: [session, path]

rprompt:
  - style: rounded
    segments: [git, time]
```

If you need full control, use `leading_style` and `trailing_style` instead of `style`.

## Step 4: Add a Transient Prompt

A transient prompt replaces the old primary prompt after you press Enter.
Use separate segment instances so you can simplify the display.

```yaml
transient:
  - segments: [path.transient]

rtransient:
  - segments: [git.transient, time.transient]

path.transient:
  foreground: lightWhite
  background: transparent
  template: " {{ .Folder }} "

git.transient:
  foreground: lightWhite
  background: transparent
  template: " {{ .HEAD }} "
  options:
    fetch_status: true

time.transient:
  foreground: darkGray
  background: transparent
  template: " {{ .CurrentDate | date \"15:04\" }} "
```

## Step 5: Centralize Colors with a Palette

```yaml
palette:
  fg: "#e6edf3"
  bg_path: "#1f6feb"
  bg_session: "#d29922"
  bg_clean: "#238636"
  bg_dirty: "#d29922"
  bg_time: "#6e7681"

session:
  foreground: p:bg_path
  background: p:bg_session
  template: " {{ .UserName }} "

path:
  foreground: p:fg
  background: p:bg_path
  template: " {{ .Path }} "

git:
  foreground: black
  background: p:bg_clean
  background_templates:
    - "{{ if or (.Working.Changed) (.Staging.Changed) }}p:bg_dirty{{ end }}"
  template: " {{ .HEAD }} "

time:
  foreground: p:fg
  background: p:bg_time
  template: " {{ .CurrentDate | date \"15:04\" }} "
```

## Step 6: Tune Slow Segments

When a segment is slow, let it stream and show a pending state:

```yaml
daemon_timeout: 100
render_pending_icon: " "
render_pending_background: darkGray

git:
  cache:
    duration: 30s
    strategy: folder
  timeout: 2000
  render_pending_icon: " "
  options:
    fetch_status: true
```

## Step 7: Validate by Rendering

Use `prompto render` directly while iterating:

```bash
prompto render --shell=zsh --pwd="$PWD" --terminal-width=120
```

## A Practical Full Example

```yaml
cursor_padding: true
render_pending_icon: " "
render_pending_background: darkGray
vim-mode:
  enabled: true
  cursor_shape: true

palette:
  fg: "#e6edf3"
  black: "#1b1f24"
  yellow: "#d29922"
  blue: "#1f6feb"
  green: "#238636"
  orange: "#db6d28"
  gray: "#6e7681"

prompt:
  - style: rounded
    segments: [session, path]

rprompt:
  - style: rounded
    segments: [git, vim, time]

transient:
  - segments: [path.transient]

rtransient:
  - style: rounded
    segments: [git.transient, time.transient]

session:
  foreground: p:black
  background: p:yellow
  template: " {{ .UserName }} "

path:
  foreground: p:fg
  background: p:blue
  template: " {{ .Path }} "

git:
  foreground: p:black
  background: p:green
  background_templates:
    - "{{ if or (.Working.Changed) (.Staging.Changed) }}p:orange{{ end }}"
  template: " {{ .HEAD }} "
  cache:
    duration: 30s
    strategy: folder
  options:
    fetch_status: true

vim:
  foreground: p:fg
  background_templates:
    - "{{ if .Normal }}red{{ end }}"
    - "{{ if .Visual }}magenta{{ end }}"
    - "{{ if .Insert }}blue{{ end }}"
  template: " {{ .Mode }} "

time:
  foreground: p:fg
  background: p:gray
  template: " {{ .CurrentDate | date \"15:04\" }} "

path.transient:
  foreground: p:fg
  background: transparent
  template: " {{ .Folder }} "

git.transient:
  foreground: p:fg
  background: transparent
  template: " {{ .HEAD }} "
  options:
    fetch_status: true

time.transient:
  foreground: p:gray
  background: transparent
  template: " {{ .CurrentDate | date \"15:04\" }} "
```
