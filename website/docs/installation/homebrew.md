<!-- markdownlint-disable-next-line MD041 -->
A [Homebrew][brew] Formula and Cask are available for easy installation.

```bash
brew install jandedobbeleer/prompto/prompto
```

Updating is done via:

```bash
brew update && brew upgrade prompto
```

:::tip
In case you see [strange behaviour][strange] in your shell, reload it after upgrading Prompto.
For example in zsh:

```bash
brew update && brew upgrade && exec zsh
```

:::

[brew]: https://brew.sh
[strange]: https://github.com/po1o/prompto/issues/1287
