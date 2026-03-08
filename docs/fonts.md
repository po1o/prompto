---
title: Fonts
description: Install a Nerd Font and configure your terminal so prompto glyphs render correctly.
---

## Why Fonts Matter

Many `prompto` themes use Powerline separators and Nerd Font glyphs.
If your terminal does not use a compatible font, you will see rectangles, missing icons, or broken separators.

## Recommended Choice

A Nerd Font is the safest default.
`Meslo` is a practical starting point and is easy to install with the built-in CLI.

## Install a Font with prompto

List and install through the built-in font command:

```bash
prompto font install
```

Install `Meslo` directly:

```bash
prompto font install meslo
```

When running as administrator or root, the font is installed system-wide.
As a regular user, it is installed into the user font directory.

## Host vs Container vs WSL

Fonts are a terminal UI concern.
If you use WSL, a remote session, or a container, the font must still be installed on the host machine running the
terminal emulator.

## Configure Common Terminals

### Windows Terminal

Set the font family in `settings.json`:

```json
{
  "profiles": {
    "defaults": {
      "font": {
        "face": "MesloLGM Nerd Font"
      }
    }
  }
}
```

### Visual Studio Code

Set the integrated terminal font family:

```json
"terminal.integrated.fontFamily": "MesloLGM Nerd Font"
```

### Apple Terminal

Open the terminal profile settings and choose the installed Nerd Font for the profile you actually use.

### iTerm2

Open `Settings > Profiles > Text` and set the profile font to the Nerd Font you installed.

## Minimal Themes

Themes with `minimal` in the filename are a better starting point when you do not want Nerd Font icons.
See [Themes](./themes.md).
