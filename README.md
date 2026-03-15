<!-- markdownlint-disable MD041 -->
![prompto logo](./.github/assets/prompto-logo_256.png)
<!-- markdownlint-enable MD041 -->

# prompto

## What This Is

`prompto` is a fork I built mostly for personal use.

It sits on the shoulders of a giant:

- [oh-my-posh website](https://ohmyposh.dev/)
- [oh-my-posh GitHub repository](https://github.com/JanDeDobbeleer/oh-my-posh)

This fork is a continuation of ideas explored in
[oh-my-posh PR #7244](https://github.com/JanDeDobbeleer/oh-my-posh/pull/7244).

## Big Disclaimer

If you are looking for an almost universal, top-notch prompt system, you should use
[oh-my-posh](https://ohmyposh.dev/), not this fork.

That is intentional.
I do not want to steal users from oh-my-posh or lure people away from it.

## Why This Fork Exists

Not pursuing this path in oh-my-posh was the right approach.

oh-my-posh has different goals: broad compatibility, a stable user experience, and a configuration model that serves a
large existing user base.

`prompto` exists so I can take a different direction without forcing that direction upstream.

This is not a drop-in replacement for oh-my-posh and it is not trying to compete with it.

## Compatibility

`prompto` configuration is purposely incompatible with oh-my-posh configuration.

That incompatibility is deliberate.
It exists in part to avoid creating a casual migration path for people who should simply keep using oh-my-posh.

If you are an oh-my-posh user and you are specifically interested in streaming capabilities, look at
[oh-my-posh experimental streaming](https://ohmyposh.dev/docs/experimental/streaming).
That is the better path for oh-my-posh users.

## Shell Support

| Shell | Status | Notes |
| --- | --- | --- |
| `zsh` | supported | fully tested on macOS and Linux |
| `fish` | supported | not fully tested yet |
| `powershell` / `pwsh` | supported | not fully tested yet |
| `bash` | partial | limited by shell behavior and may be dropped |

## Documentation

There is no separate website for this fork.
The documentation lives in this repository.

- [Documentation index](./docs/README.md)
- [Installation](./docs/installation.md)
- [Shell initialization](./docs/shell-init.md)
- [Configuration](./docs/configuration.md)
- [Configuration reference](./docs/configuration/reference.md)
- [Segment reference](./docs/segments/README.md)
- [FAQ](./docs/faq.md)

## Project Status

This is a personal project and it is optimized for my workflow first.

If you need a mature, broadly compatible, general-purpose prompt system, use
[oh-my-posh](https://ohmyposh.dev/).
