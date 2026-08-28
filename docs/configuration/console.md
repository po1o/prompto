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

If the variant does not exist, `prompto` falls back to `config.yaml` — which on
a console is the broken-looking prompt this page exists to avoid. The fallback
is deliberate: a missing variant means "not configured", and the alternatives
are worse. Refusing to render leaves you with no prompt at all, and degrading
`config.yaml` automatically would mean guessing which glyphs your console font
carries and remapping colors silently.

So the feature is opt-in, and creating the file is the whole opt-in.

## Bundled Themes

A bundled theme can ship its own console variant as `<name>.console.prompto.yaml`, and `prompto config set`
installs both files when it does.

Every bundled theme currently ships one, so `prompto config set <theme>` gives you a usable console prompt
without any extra work. The variants are hand-maintained, though, and nothing generates them: a theme added
without one falls back to `config.yaml` on the console exactly as described above, which is the unreadable
prompt this feature exists to avoid. See [Themes](../themes.md#console-variants).

## Separators On A Console

A console config is recognised by its name — the same `.console` marker that
[Resolve](#naming-rule) uses to find it. Nothing inside the file declares it.

Two things follow. Separator aliases compile to ASCII instead of Nerd Font
glyphs, and a separator on a segment with no background is drawn in the
segment's `foreground` instead of vanishing — console themes leave backgrounds
transparent, and the usual rule would draw the separator in nothing at all.

| alias | graphical | console |
| --- | --- | --- |
| `powerline` | `\ue0b2` `\ue0b0` | `<` `>` |
| `powerline_thin` | `\ue0b3` `\ue0b1` | `<` `>` |
| `rounded` | `\ue0b6` `\ue0b4` | `(` `)` |
| `rounded_thin` | `\ue0b7` `\ue0b5` | `(` `)` |
| `slant` | `\ue0ba` `\ue0bc` | `\` `/` |
| `block` | `\ue0b8` `\ue0be` | `[` `]` |
| `flame`, `pixel`, `lego` | — | dropped |

The last three have no ASCII that reads as the same shape, so they are dropped
rather than replaced by something misleading. A literal Nerd Font glyph written
into `trailing_separator` is translated the same way, since it is just as
unreadable on a console as the alias that produces it.

Because the name carries the decision, one daemon can serve a console session
and a desktop session at once: each render passes its own config path. See
[`separator_foreground`](./segments.md#separator_foreground) to color a
separator explicitly.

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

**Drop Nerd Font icons.** Separators are handled for you once `console: true`
is set, but icons inside templates are not:

- no `branch_icon`, `github_icon`, or `git_icon`; set `fetch_upstream_icon:
  false`
- ASCII replacements for typographic characters: `..` for `…`, `↑`/`↓` for
  `⇡`/`⇣`

Separators are written exactly as in the graphical theme — the alias is
translated, and `separator_foreground` colors it against a transparent
background:

```yaml
prompt:
  - segments: [path]

path:
  style: powerline
  foreground: lightWhite
  background: transparent
  separator_foreground: red
  template: "{{ .Path }} "
```

That renders `~ >` with a red `>`. Writing the ASCII out by hand with
`trailing_separator: ">"` still works, and is what you want for a shape the
alias table does not cover.

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
