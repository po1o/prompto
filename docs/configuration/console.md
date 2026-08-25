# Console Config

## The Problem

A prompt tuned for a graphical terminal does not survive a text console.

The Linux virtual console (what you get on `Ctrl+Alt+F2`, or on a machine with no
desktop session) has two hard limits:

- **No Nerd Font.** The console font holds a few hundred glyphs. Powerline
  separators and Nerd Font icons render as blanks or rectangles.
- **16 colors.** Truecolor escape sequences are ignored, so hex colors either
  collapse to something unintended or drop out entirely.

Rather than degrade one config with conditionals, `prompto` lets you keep a
second config written for those limits.

## How It Works

Put a `config.console.yaml` next to your `config.yaml`:

```text
~/.config/prompto/
├── config.yaml          # terminal emulator
└── config.console.yaml  # text console
```

When `prompto init` runs on a console and the variant exists, that variant is
used instead of `config.yaml`. Nothing else changes: same segment names, same
layout keys, same templates.

If the variant does not exist, `prompto` uses `config.yaml` as usual. Adding the
file is the only thing you have to do to opt in.

## Bundled Themes

A bundled theme can ship its own console variant as `<name>.console.prompto.yaml`, and `prompto config set`
installs both files when it does. The `polo` theme is one such theme. See
[Themes](../themes.md#console-variants).

## Naming Rule

The variant is the config path with `.console` inserted before the extension.
This applies to any config path, not only the default one:

| Requested config | Console variant |
| --- | --- |
| `~/.config/prompto/config.yaml` | `~/.config/prompto/config.console.yaml` |
| `--config ~/themes/mine.yaml` | `~/themes/mine.console.yaml` |
| `--config ~/themes/mine.yml` | `~/themes/mine.console.yml` |

## Detection

A session counts as a console when:

| Condition | Result |
| --- | --- |
| `PROMPTO_CONSOLE=1` | console, whatever `TERM` says |
| `PROMPTO_CONSOLE=0` | not a console, whatever `TERM` says |
| `TERM=linux` | console |
| anything else | not a console |

`PROMPTO_CONSOLE` exists so you can force the decision: to preview the console
config from your normal terminal, or to opt in a serial console or framebuffer
terminal that reports some other `TERM`.

Preview the console config without leaving your terminal emulator:

```bash
PROMPTO_CONSOLE=1 prompto init zsh --print
```

## Resolution Happens Once, At Init

The choice is made by `prompto init`, which runs inside the shell itself and so
sees that session's `TERM`. The resolved path is baked into the init script and
passed back on every later render.

That matters when a console session and a desktop terminal session are open at
the same time: each one carries its own config path, so they can share a daemon
without either overriding the other.

Because the decision is made at init, opening a new shell is what picks up a
newly added `config.console.yaml`.

## Writing One

Two rules cover most of it.

**Use ANSI color names, not hex.** These map onto the 16 colors a console
actually has:

`black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`,
`darkGray`, `lightRed`, `lightGreen`, `lightYellow`, `lightBlue`,
`lightMagenta`, `lightCyan`, `lightWhite`

**Drop Powerline and Nerd Font glyphs.** That means:

- `style: plain` with `background: transparent` instead of `powerline` or
  `diamond` segments
- no `style: rounded` and no other separator alias on prompt lines, since every
  alias is a Powerline glyph
- ASCII separators written into the templates instead, e.g. a `>` closing the
  left prompt and a `<` opening the right one
- no `branch_icon`, `github_icon`, or `git_icon`; set `fetch_upstream_icon:
  false`
- ASCII replacements for typographic characters: `..` for `…`, `↑`/`↓` for
  `⇡`/`⇣`

A left prompt built this way:

```yaml
prompt:
  - segments: [path]

path:
  style: plain
  foreground: lightWhite
  background: transparent
  template: "{{ .Path }} <red>></>"
```

## Checking Your Config Is Console-Safe

List every non-ASCII character in the file and confirm you recognize each one:

```bash
grep -oP '[^\x00-\x7F]' ~/.config/prompto/config.console.yaml | sort -u
```

Nerd Font glyphs written as `\uXXXX` escapes stay ASCII in the file and will not
show up that way, so grep for those too:

```bash
grep -o '\\u[0-9a-f]\{4\}' ~/.config/prompto/config.console.yaml | sort -u
```

Both should come back empty, or contain only characters your console font
carries. Arrows such as `↑` and `↓` are usually present; box drawing and
typographic punctuation usually are not.

To confirm no truecolor slipped in, render the prompt and look at the escape
sequences. Console-safe output uses only `30`–`37`, `90`–`97`, `39`, `49`, and
attribute codes; a `38;2;` or `48;2;` means a hex color survived:

```bash
prompto render --config ~/.config/prompto/config.console.yaml --escape=false | cat -v
```
