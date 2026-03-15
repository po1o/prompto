<!-- markdownlint-disable MD041 -->
![prompto logo](./.github/assets/prompto-logo_256.png)
<!-- markdownlint-enable MD041 -->

# Prompto

## What This Is

`prompto` is a fork of [oh-my-posh](https://github.com/JanDeDobbeleer/oh-my-posh) that I built mostly for personal use. It is a continuation of ideas explored in [oh-my-posh PR #7244](https://github.com/JanDeDobbeleer/oh-my-posh/pull/7244).

Because, it comes from [oh-my-posh](https://ohmyposh.dev/), it sits on the shoulders of a giant. However, it is now massively different from oh-my-posh.

## Why This Fork Exists

Not pursuing [PR #7244](https://github.com/JanDeDobbeleer/oh-my-posh/pull/7244) in oh-my-posh was the **right** approach.

`prompto` exists so I can take a different direction without forcing that direction upstream.

oh-my-posh has different goals: broad compatibility, a stable user experience, and a configuration model that serves a
large existing user base.

## Big Disclaimer

If you need a mature, broadly compatible, general-purpose, almost universal, top-notch prompt system, use [oh-my-posh](https://ohmyposh.dev/), not this fork.

This is not a drop-in replacement for oh-my-posh and it is not trying to compete with it.
That is intentional. I do not want to steal users from oh-my-posh or lure people away from it.

`prompto` configuration is purposely incompatible with oh-my-posh configuration.

That incompatibility is deliberate.
It exists in part to avoid creating a casual migration path for people who should simply keep using oh-my-posh.

If you are an oh-my-posh user and you are specifically interested in streaming asynchronous capabilities, look at
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
