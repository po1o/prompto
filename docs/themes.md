# Themes

## Where Themes Live

Bundled themes are stored in [`themes/`](../themes).
They are plain YAML files, and prompto also compiles them into the binary for `config list` and `config set`.

Examples:

- [`themes/agnoster.minimal.prompto.yaml`](../themes/agnoster.minimal.prompto.yaml)
- [`themes/tokyo.prompto.yaml`](../themes/tokyo.prompto.yaml)
- [`themes/powerlevel10k_modern.prompto.yaml`](../themes/powerlevel10k_modern.prompto.yaml)

## Recommended Workflow

1. Pick a theme that is visually close to what you want.
2. Write it to the default config path.
3. Point shell init at that local file.
4. Edit the copy instead of editing the theme in-place.

List the bundled themes:

```bash
prompto config list
```

Write one to the default config path. If a config already exists there, prompto asks before overwriting it:

```bash
prompto config set tokyo
```

Then initialize your shell against that file:

```bash
eval "$(prompto init --config ~/.config/prompto/config.yaml)"
```

## Console Variants

A theme may ship a second file for the Linux virtual console, named with `.console` before the theme suffix:

```text
themes/polo.prompto.yaml          # terminal emulator
themes/polo.console.prompto.yaml  # text console
```

The variant is not a theme of its own. It never appears in `prompto config list`, and it cannot be selected by
name: it belongs to `polo`. Every `<name>.console.prompto.yaml` must have a matching `<name>.prompto.yaml`, and
the theme generator fails if one does not.

When a theme has a variant, `config set` installs both:

```console
$ prompto config set polo
wrote ~/.config/prompto/config.yaml
wrote ~/.config/prompto/config.console.yaml (console variant)
```

Most themes have no variant, in which case only `config.yaml` is written. Switching from a theme that has one to
a theme that does not leaves the old `config.console.yaml` in place — `prompto` warns about this rather than
deleting a file you may have written yourself:

```console
$ prompto config set tokyo
wrote ~/.config/prompto/config.yaml
warning: ~/.config/prompto/config.console.yaml is left over from another config and still applies on the
console; remove it to use "tokyo" there
```

See [Console config](./configuration/console.md) for how the variant is selected at runtime and how to write one.

## Render a Theme Preview

If you want a quick preview image of the config you are currently using:

```bash
prompto config image --output ./theme-preview.png
```

## Theme Selection Advice

- Pick a `minimal` theme when you do not want icon-heavy glyphs.
- Pick a theme with a clear palette if you plan to reuse colors across many segments.
- Prefer local theme files over generated or remote setups when performance and debuggability matter.

## What to Edit First

Common first edits:

- `prompt` and `rprompt` segment order
- `palette` or `palettes`
- `path`, `git`, `time`, and `session` templates
- `transient` and `rtransient`

The editing model is documented in [Configuration](./configuration.md).
