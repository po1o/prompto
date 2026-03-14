# Colors

## Supported Color Forms

`prompto` accepts several color formats.

### Hex colors

```yaml
foreground: "#e6edf3"
background: "#1f6feb"
```

### ANSI color names

Supported names:

- `black`
- `red`
- `green`
- `yellow`
- `blue`
- `magenta`
- `cyan`
- `white`
- `default`
- `darkGray`
- `lightRed`
- `lightGreen`
- `lightYellow`
- `lightBlue`
- `lightMagenta`
- `lightCyan`
- `lightWhite`

### 256-color indexes

```yaml
foreground: "214"
background: "238"
```

## Color Keywords

These keywords are resolved dynamically:

- `transparent`
- `accent`
- `foreground`
- `background`
- `parentForeground`
- `parentBackground`

Examples:

```yaml
text.separator:
  type: text
  foreground: parentBackground
  background: transparent
  template: ""
```

## Palette

Use `palette` to centralize color names.
Palette references use the `p:` prefix.

```yaml
palette:
  fg: "#e6edf3"
  bg_path: "#1f6feb"
  bg_git_clean: "#238636"
  bg_git_dirty: "#d29922"

path:
  foreground: p:fg
  background: p:bg_path
  template: " {{ .Path }} "

git:
  foreground: black
  background: p:bg_git_clean
  background_templates:
    - "{{ if or (.Working.Changed) (.Staging.Changed) }}p:bg_git_dirty{{ end }}"
  template: " {{ .HEAD }} "
```

## Recursive Palette References

A palette entry can point to another palette entry.

```yaml
palette:
  blue_base: "#1f6feb"
  primary_bg: p:blue_base
```

## Invalid Palette References

If a palette reference cannot be resolved, the renderer falls back as if the color were transparent.
Treat missing palette keys as configuration bugs.

## Conditional Palettes

Use `palettes` when you want runtime palette selection.

```yaml
palettes:
  template: "{{ if eq .Shell \"pwsh\" }}windows{{ else }}unix{{ end }}"
  list:
    unix:
      fg: "#e6edf3"
      accent_bg: "#1f6feb"
    windows:
      fg: "#ffffff"
      accent_bg: "#005fb8"
```

You can still define a top-level `palette` alongside `palettes`.
The base `palette` fills missing keys in the selected named palette.

## Color Templates

Use `foreground_templates` and `background_templates` for conditional colors.
The first non-empty rendered template wins.
If none match, the base `foreground` or `background` value is used.

```yaml
git:
  foreground: black
  background: green
  background_templates:
    - "{{ if or (.Working.Changed) (.Staging.Changed) }}yellow{{ end }}"
    - "{{ if gt .Behind 0 }}red{{ end }}"
  template: " {{ .HEAD }} "
```

## Cycle

`cycle` applies a repeating list of foreground/background pairs to rendered segments.
This is useful for intentionally alternating colors.

```yaml
cycle:
  - foreground: black
    background: lightBlue
  - foreground: black
    background: lightGreen
  - foreground: black
    background: lightMagenta
```

When `cycle` is active, rendered segments consume the next color pair in order.

## Color Advice

- Use a palette for any config larger than a few segments.
- Keep foreground contrast high enough for terminal readability.
- Use background templates for state changes such as dirty git status or failing commands.
- Reserve `accent` for cases where platform integration is actually useful.
