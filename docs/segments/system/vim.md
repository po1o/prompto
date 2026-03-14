# Vim Mode

## Segment Type

`vim`

## What

Display the current vim editing mode reported by the shell integration.

This segment is only useful when the top-level `vim-mode` feature is enabled and your shell integration supports vim
mode reporting. See [Extras and shell features](../../configuration/extras.md#vim-mode).

## Sample Configuration

```yaml
vim-mode:
  enabled: true
  cursor_shape: true
  cursor_blink: false

prompt:
  - segments: [vim, path]

vim:
  type: vim
  style: diamond
  foreground: white
  background: "#4c566a"
  background_templates:
    - "{{ if .Insert }}#4caf50{{ end }}"
    - "{{ if .Normal }}#ffb300{{ end }}"
    - "{{ if .Visual }}#7e57c2{{ end }}"
    - "{{ if .Replace }}#ef5350{{ end }}"
  leading_diamond: ""
  trailing_diamond: ""
```

## Behavior

- mode changes are repaint-driven, so the segment updates without restarting slow async segment work
- primary, right, transient, and right-transient prompts can all reference the same `vim` segment definition
- if `vim-mode.enabled` is off, the shell will not report mode changes and this segment will not be useful

## Options

This segment has no segment-specific `options` entries.
Use ordinary segment fields such as `template`, `style`, `foreground`, `background`, and color templates.

## Template

### Default Template

```template
{{ if .Insert }} INSERT {{ end }}{{ if .Normal }} NORMAL {{ end }}{{ if .Visual }} VISUAL {{ end }}{{ if .Replace }} REPLACE {{ end }}
```

### Properties

- `.Insert`
  - Type: `boolean`
  - Description: true when the shell reports insert mode
- `.Normal`
  - Type: `boolean`
  - Description: true when the shell reports normal mode
- `.Visual`
  - Type: `boolean`
  - Description: true when the shell reports visual mode
- `.Replace`
  - Type: `boolean`
  - Description: true when the shell reports replace mode

## Related Pages

- [Extras and shell features](../../configuration/extras.md#vim-mode)
- [Shell initialization](../../shell-init.md)
